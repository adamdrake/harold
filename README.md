# **This repository has been archived**

# Hello, Harold!

Harold is a twitter bot which can manage tweeting from feeds at specified intervals, keeping your timeline groomed by removing tweets after a certain amount of time, of even removing all of your tweets or DMs entirely.

Harold takes a config file in TOML format (with -config command line flag), and whether or not you want to run in `botmode`, `deletedm`, or `deletetimeline`.

```
harold -botmode -config=/path/to/config.toml
```

# Twitter API

You will need to create a new Twitter application and generate API keys.  Harold assumes the following environment variables are set:

```
TWITTER_CONSUMER_KEY
TWITTER_CONSUMER_SECRET
TWITTER_ACCESS_TOKEN
TWITTER_ACCESS_TOKEN_SECRET
```

One way to do this would be to get the API keys and then put them into a file, which you can then source before running Harold.

```
export TWITTER_CONSUMER_KEY=xxxxx
.
.
.
```

Then just `source keys.env` or whatever you have named the file containing the keys, or you will get a response from Twitter saying you are unauthorized.

# Configuring Harold

The config file has a few different parameters, and an example is below.

```
cleanTimeline = true
maxTweetAge = "24h"
cleanTimelineEvery = "1h"

[feeds]
[feeds.aadrakecom]
url = "https://aadrake.com/index.xml"
hashTags = "#tech #datascience #ai #ml #techto"
maxTweetInterval = "12h"

[feeds.datamultireddit]
url = "https://www.reddit.com/user/adrake/m/data/.rss"
hashTags = "#data #bigdata #ai #ml"
maxTweetInterval = "12h"

[feeds.datatau]
url = "http://www.datatau.com/rss"
hashTags = "#data #bigdata #ai #ml #datascience"
maxTweetInterval = "12h"

[feeds.cryptocurrencynews]
url = "https://cryptocurrencynews.com/feed/"
hashTags = "#cryptocurrency #blockchain #btc #eth #xrp #xrb"
maxTweetInterval = "12h"
```

`cleanTimeline` determines whether or not Harold will remove tweets older than `maxTweetAge` from your timeline when running in `botmode`, which will be done at frequency `cleanTimelineEvery`.

Each entry in the feeds section takes a `url`, a list of hashtags in the `hashTags` parameter which you want appended to tweets from that feed, and a `maxTweetInterval`, which is the longest time between tweets for that feed.  The frequency of tweets from a given feed is uniformly distributed between now and now + `maxTweetInterval`.

