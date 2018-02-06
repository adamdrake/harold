package main

import (
	"errors"
	"flag"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/ChimeraCoder/anaconda"
	log "github.com/Sirupsen/logrus"
	"github.com/mmcdole/gofeed"
	"github.com/spf13/viper"
)

var (
	logger = log.New()
)

func randomItemFromFeed(feedURL string) (gofeed.Item, error) {
	// Reddit (for example) will return an HTTP 429 if no User-Agent string is provided...
	userAgent := "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:58.0) Gecko/20100101 Firefox/58.0"
	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return gofeed.Item{}, errors.New("Could not construct http request")
	}
	req.Header.Add("User-Agent", userAgent)
	client := http.Client{}
	resp, err := client.Do(req)

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	resp.Body.Close() // Close response body in order to avoid leaks
	if err != nil {
		return gofeed.Item{}, errors.New("Could not parse feed")

	}
	if feed == nil {
		return gofeed.Item{}, errors.New("Gofeed returned nil feed!")

	}
	if len(feed.Items) < 1 {
		return gofeed.Item{}, errors.New("Empty feed returned from URL")
	}

	idx := rand.Intn(len(feed.Items))
	return *feed.Items[idx], nil

}

func randomTweetFromFeed(api *anaconda.TwitterApi, feedURL, hashtags string, frequency time.Duration) {

	for {
		item, err := randomItemFromFeed(feedURL)
		if err != nil {
			log.Error(err)
			continue
		}

		tweetText := item.Title + " " + item.Link + " " + hashtags

		tweet, err := api.PostTweet(tweetText, url.Values{})
		if err != nil {
			log.Error("Could not post tweet: ", tweet, " ", err)
			time.Sleep(60 * time.Second) // In case of failure, don't hammer any sites
			continue
		}

		log.Info("TWEETED ", tweet.Text)
		waitTime := time.Duration(rand.Intn(int(frequency.Seconds()))) * time.Second
		log.Info("Sleeping ", waitTime, " before tweeting again...")
		time.Sleep(waitTime)
	}
}

func getTimeline(api *anaconda.TwitterApi) ([]anaconda.Tweet, error) {
	args := url.Values{}
	args.Add("count", "3200")       // Twitter only returns most recent 20 tweets by default, so override
	args.Add("include_rts", "true") // When using count argument, RTs are excluded, so include them as recommended
	timeline, err := api.GetUserTimeline(args)
	if err != nil {
		return make([]anaconda.Tweet, 0), err
	}
	return timeline, nil
}

func deleteTweets(tweets []anaconda.Tweet, api *anaconda.TwitterApi) error {
	for _, t := range tweets {
		_, err := api.DeleteTweet(t.Id, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteAllTweets(api *anaconda.TwitterApi) error {
	timeline, err := getTimeline(api)
	for len(timeline) > 1 {
		err = deleteTweets(timeline, api)
		if err != nil {
			return err
		}
		timeline, err = getTimeline(api)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteFromTimeline(api *anaconda.TwitterApi, ageLimit, sleepTime time.Duration) {
	for {
		timeline, err := getTimeline(api)
		if err != nil {
			log.Error("Could not get timeline")
		}
		for _, t := range timeline {
			createdTime, err := t.CreatedAtTime()
			if err != nil {
				log.Error("Couldn't parse time ", err)
			} else {
				if time.Since(createdTime) > ageLimit {
					_, err := api.DeleteTweet(t.Id, true)
					log.Info("DELETED: Age - ", time.Since(createdTime).Round(1*time.Minute), " - ", t.Text)
					if err != nil {
						log.Error("Failed to delete! ", err)
					}
				}
			}
		}
		time.Sleep(sleepTime)
	}
}

func getDMs(api *anaconda.TwitterApi) ([]anaconda.DirectMessage, error) {
	args := url.Values{}
	args.Add("count", "200")
	dmsRecvd, err := api.GetDirectMessages(args)
	if err != nil {
		return make([]anaconda.DirectMessage, 0), err
	}
	dmsSent, err := api.GetDirectMessagesSent(args)
	if err != nil {
		return make([]anaconda.DirectMessage, 0), err
	}
	return append(dmsRecvd, dmsSent...), nil
}

func destroyDMs(api *anaconda.TwitterApi) {
	dms, err := getDMs(api)
	if err != nil {
		log.Error(err)
	}
	if len(dms) > 0 {
		for _, dm := range dms {
			m, err := api.DeleteDirectMessage(dm.Id, true)
			if err != nil {
				log.Error("Could not delete DM ", err, m)
			}
			log.Info("Deleted DM from ", m.CreatedAt, " ", m.Text)
			time.Sleep(1 * time.Second)
		}
		destroyDMs(api)
	}

}

type config struct {
	CleanTimeline      bool
	MaxTweetAge        time.Duration
	CleanTimelineEvery time.Duration
	Feeds              map[string]feed
}

type feed struct {
	URL, Hashtags    string
	MaxTweetInterval time.Duration
}

func main() {
	rand.Seed(time.Now().UnixNano())
	var deleteDMs = flag.Bool("deletedm", false, "Only delete direct messages, then exit")
	var configPath = flag.String("config", "config.toml", "Path to the configuration file (in TOML format)")
	var deleteTimeline = flag.Bool("deletetimeline", false, "Delete all tweets in timeline, then exit")
	var botMode = flag.Bool("botmode", false, "Run in bot mode (tweet from feeds, keep timeline groomed, etc.)")
	flag.Parse()

	viper.SetConfigFile(*configPath)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	var c config
	err = viper.Unmarshal(&c)
	if err != nil {
		panic(err)
	}
	viper.AutomaticEnv()

	anaconda.SetConsumerKey(viper.GetString("TWITTER_CONSUMER_KEY"))
	anaconda.SetConsumerSecret(viper.GetString("TWITTER_CONSUMER_SECRET"))
	api := anaconda.NewTwitterApi(viper.GetString("TWITTER_ACCESS_TOKEN"), viper.GetString("TWITTER_ACCESS_TOKEN_SECRET"))
	api.SetLogger(anaconda.BasicLogger)

	fmter := new(log.TextFormatter)
	fmter.FullTimestamp = true
	log.SetFormatter(fmter)
	log.SetLevel(log.InfoLevel)

	if *deleteDMs {
		destroyDMs(api)
		os.Exit(0)
	}
	if *deleteTimeline {
		deleteAllTweets(api)
		os.Exit(0)
	}
	if *botMode {
		if len(c.Feeds) > 0 {
			for _, f := range c.Feeds {
				go randomTweetFromFeed(api, f.URL, f.Hashtags, f.MaxTweetInterval)
			}
		}
		if c.CleanTimeline {
			go deleteFromTimeline(api, c.MaxTweetAge, c.CleanTimelineEvery)
		}
		var wg sync.WaitGroup
		wg.Add(1)
		wg.Wait()
	}
}
