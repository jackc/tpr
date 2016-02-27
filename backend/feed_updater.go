package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/jackc/tpr/backend/data"
	"golang.org/x/net/html/charset"
	log "gopkg.in/inconshreveable/log15.v2"
	"io/ioutil"
	"net/http"
	"time"
)

type FeedUpdater struct {
	client                   *http.Client
	maxConcurrentFeedFetches int
	repo                     repository
	logger                   log.Logger
}

func NewFeedUpdater(repo repository, logger log.Logger) *FeedUpdater {
	feedUpdater := &FeedUpdater{}
	feedUpdater.repo = repo
	feedUpdater.logger = logger
	feedUpdater.client = &http.Client{Timeout: 60 * time.Second}
	feedUpdater.maxConcurrentFeedFetches = 25
	return feedUpdater
}

func (u *FeedUpdater) KeepFeedsFresh() {

	for {
		startTime := time.Now()

		if staleFeeds, err := u.repo.GetFeedsUncheckedSince(startTime.Add(-10 * time.Minute)); err == nil {
			u.logger.Info("GetFeedsUncheckedSince succeeded", "n", len(staleFeeds))

			staleFeedChan := make(chan data.Feed)
			finishChan := make(chan bool)

			worker := func() {
				for feed := range staleFeedChan {
					u.RefreshFeed(feed)
				}
				finishChan <- true
			}

			for i := 0; i < u.maxConcurrentFeedFetches; i++ {
				go worker()
			}

			for _, sf := range staleFeeds {
				staleFeedChan <- sf
			}
			close(staleFeedChan)

			for i := 0; i < u.maxConcurrentFeedFetches; i++ {
				<-finishChan
			}

		} else {
			u.logger.Error("GetFeedsUncheckedSince failed", "error", err)
		}

		sleepUntil(startTime.Add(time.Minute))
	}
}

// sleepUntil sleeps until t. If t is in the past it returns immediately
func sleepUntil(t time.Time) {
	time.Sleep(t.Sub(time.Now()))
}

type rawFeed struct {
	url  string
	body []byte
	etag data.String
}

func (u *FeedUpdater) fetchFeed(feedURL string, etag data.String) (*rawFeed, error) {
	feed := &rawFeed{url: feedURL}

	req, err := http.NewRequest("GET", feed.url, nil)
	if etag.Status == data.Present {
		req.Header.Add("If-None-Match", etag.Value)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		feed.body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("Unable to read response body: %v", err)
		}

		feed.etag = newStringFallback(resp.Header.Get("Etag"), data.Null)

		return feed, nil
	case 304:
		return nil, nil
	default:
		return nil, fmt.Errorf("Bad HTTP response: %s", resp.Status)
	}
}

func (u *FeedUpdater) RefreshFeed(staleFeed data.Feed) {
	rawFeed, err := u.fetchFeed(staleFeed.URL.Value, staleFeed.ETag)
	if err != nil {
		u.logger.Error("fetchFeed failed", "url", staleFeed.URL.Value, "error", err)
		u.repo.UpdateFeedWithFetchFailure(staleFeed.ID.Value, err.Error(), time.Now())
		return
	}
	// 304 unchanged
	if rawFeed == nil {
		u.logger.Info("fetchFeed 304 unchanged", "url", staleFeed.URL.Value)
		u.repo.UpdateFeedWithFetchUnchanged(staleFeed.ID.Value, time.Now())
		return
	}

	feed, err := parseFeed(rawFeed.body)
	if err != nil {
		u.logger.Error("parseFeed failed", "url", staleFeed.URL.Value, "error", err)
		u.repo.UpdateFeedWithFetchFailure(staleFeed.ID.Value, fmt.Sprintf("Unable to parse feed: %v", err), time.Now())
		return
	}

	u.logger.Info("refreshFeed succeeded", "url", staleFeed.URL.Value, "id", staleFeed.ID.Value)
	u.repo.UpdateFeedWithFetchSuccess(staleFeed.ID.Value, feed, rawFeed.etag, time.Now())
}

type parsedItem struct {
	url             string
	title           string
	publicationTime data.Time
}

func (i *parsedItem) isValid() bool {
	return i.url != "" && i.title != ""
}

type parsedFeed struct {
	name  string
	items []parsedItem
}

func (f *parsedFeed) isValid() bool {
	if f.name == "" {
		return false
	}

	for _, item := range f.items {
		if !item.isValid() {
			return false
		}
	}

	return true
}

func parseFeed(body []byte) (f *parsedFeed, err error) {
	f, err = parseRSS(body)
	if err == nil {
		return f, nil
	}

	return parseAtom(body)
}

func parseRSS(body []byte) (*parsedFeed, error) {
	type Item struct {
		Link    string `xml:"link"`
		Title   string `xml:"title"`
		Date    string `xml:"date"`
		PubDate string `xml:"pubDate"`
	}

	type Channel struct {
		Title       string `xml:"title"`
		Description string `xml:"description"`
		Item        []Item `xml:"item"`
	}

	var rss struct {
		Channel Channel `xml:"channel"`
		Item    []Item  `xml:"item"`
	}

	err := parseXML(body, &rss)
	if err != nil {
		return nil, err
	}

	var feed parsedFeed
	if rss.Channel.Title != "" {
		feed.name = rss.Channel.Title
	} else {
		feed.name = rss.Channel.Description
	}

	var items []Item
	if len(rss.Item) > 0 {
		items = rss.Item
	} else {
		items = rss.Channel.Item
	}

	feed.items = make([]parsedItem, len(items))
	for i, item := range items {
		feed.items[i].url = item.Link
		feed.items[i].title = item.Title
		if item.Date != "" {
			feed.items[i].publicationTime, _ = parseTime(item.Date)
		}
		if item.PubDate != "" {
			feed.items[i].publicationTime, _ = parseTime(item.PubDate)
		}
	}

	if !feed.isValid() {
		return nil, errors.New("Invalid RSS")
	}

	return &feed, nil
}

func parseAtom(body []byte) (*parsedFeed, error) {
	type Link struct {
		Href string `xml:"href,attr"`
	}

	type Entry struct {
		Link      Link   `xml:"link"`
		Title     string `xml:"title"`
		Published string `xml:"published"`
		Updated   string `xml:"updated"`
	}

	var atom struct {
		Title string  `xml:"title"`
		Entry []Entry `xml:"entry"`
	}

	err := parseXML(body, &atom)
	if err != nil {
		return nil, err
	}

	var feed parsedFeed
	feed.name = atom.Title
	feed.items = make([]parsedItem, len(atom.Entry))
	for i, entry := range atom.Entry {
		feed.items[i].url = entry.Link.Href
		feed.items[i].title = entry.Title
		if entry.Published != "" {
			feed.items[i].publicationTime, _ = parseTime(entry.Published)
		}
		if entry.Updated != "" {
			feed.items[i].publicationTime, _ = parseTime(entry.Updated)
		}
	}

	if !feed.isValid() {
		return nil, errors.New("Invalid Atom")
	}

	return &feed, nil
}

// Parse XML laxly
func parseXML(body []byte, doc interface{}) error {
	buf := bytes.NewBuffer(body)
	decoder := xml.NewDecoder(buf)
	decoder.CharsetReader = charset.NewReaderLabel

	decoder.Entity = xml.HTMLEntity

	return decoder.Decode(doc)
}

// Try multiple time formats one after another until one works or all fail
func parseTime(value string) (data.Time, error) {
	formats := []string{
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		time.RFC822,
		"02 Jan 2006 15:04 MST",           // RFC822 with 4 digit year
		"02 Jan 2006 15:04:05 MST",        // RFC822 with 4 digit year and seconds
		"Mon, _2 Jan 2006 15:04:05 MST",   // RFC1123 with 1-2 digit days
		"Mon, _2 Jan 2006 15:04:05 -0700", // RFC1123 with numeric time zone and with 1-2 digit days
		"Mon, _2 Jan 2006",
		"2006-01-02",
	}
	for _, f := range formats {
		t, err := time.Parse(f, value)
		if err == nil {
			return data.NewTime(t), nil
		}
	}

	return data.Time{}, errors.New("Unable to parse time")
}
