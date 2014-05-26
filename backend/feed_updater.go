package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/JackC/box"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type FeedUpdater struct {
	client                   *http.Client
	bodyResponseTimeout      time.Duration
	maxConcurrentFeedFetches int
	repo                     repository
}

func NewFeedUpdater(repo repository) *FeedUpdater {
	feedUpdater := &FeedUpdater{}
	feedUpdater.repo = repo
	transport := &http.Transport{ResponseHeaderTimeout: time.Duration(10 * time.Second)}
	feedUpdater.client = &http.Client{Transport: transport}
	feedUpdater.bodyResponseTimeout = 60 * time.Second
	feedUpdater.maxConcurrentFeedFetches = 25
	return feedUpdater
}

func (u *FeedUpdater) KeepFeedsFresh() {

	for {
		startTime := time.Now()

		if staleFeeds, err := u.repo.GetFeedsUncheckedSince(startTime.Add(-10 * time.Minute)); err == nil {
			logger.Info("tpr", fmt.Sprintf("Found %d stale feeds", len(staleFeeds)))

			staleFeedChan := make(chan Feed)
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
			logger.Error("tpr", fmt.Sprintf("repo.GetFeedsUncheckedSince failed: %v", err))
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
	etag box.String
}

func (u *FeedUpdater) fetchFeed(feedURL string, etag box.String) (*rawFeed, error) {
	feed := &rawFeed{url: feedURL}

	req, err := http.NewRequest("GET", feed.url, nil)
	if etag, ok := etag.Get(); ok {
		req.Header.Add("If-None-Match", etag)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		done := make(chan bool, 1)
		go func() {
			feed.body, err = ioutil.ReadAll(resp.Body)
			done <- true
		}()

		select {
		case <-done:
			if err != nil {
				return nil, fmt.Errorf("Unable to read response body: %v", err)
			}
		case <-time.After(u.bodyResponseTimeout):
			u.client.Transport.(*http.Transport).CancelRequest(req)
			return nil, &url.Error{Op: "Get", URL: feedURL, Err: errors.New("Timeout receiving response body")}
		}

		feed.etag.SetCoerceZero(resp.Header.Get("Etag"), box.Empty)

		return feed, nil
	case 304:
		return nil, nil
	default:
		return nil, fmt.Errorf("Bad HTTP response: %s", resp.Status)
	}
}

func (u *FeedUpdater) RefreshFeed(staleFeed Feed) {
	rawFeed, err := u.fetchFeed(staleFeed.URL.MustGet(), staleFeed.ETag)
	if err != nil {
		logger.Error("tpr", fmt.Sprintf("fetchFeed %s failed: %v", staleFeed.URL.MustGet(), err))
		u.repo.UpdateFeedWithFetchFailure(staleFeed.ID.MustGet(), err.Error(), time.Now())
		return
	}
	// 304 unchanged
	if rawFeed == nil {
		logger.Info("tpr", fmt.Sprintf("fetchFeed %s 304 unchanged", staleFeed.URL.MustGet()))
		u.repo.UpdateFeedWithFetchUnchanged(staleFeed.ID.MustGet(), time.Now())
		return
	}

	feed, err := parseFeed(rawFeed.body)
	if err != nil {
		logger.Error("tpr", fmt.Sprintf("parseFeed %s failed: %v", staleFeed.URL.MustGet(), err))
		u.repo.UpdateFeedWithFetchFailure(staleFeed.ID.MustGet(), fmt.Sprintf("Unable to parse feed: %v", err), time.Now())
		return
	}

	logger.Info("tpr", fmt.Sprintf("refreshFeed %s (%d) succeeded", staleFeed.URL.MustGet(), staleFeed.ID.MustGet()))
	u.repo.UpdateFeedWithFetchSuccess(staleFeed.ID.MustGet(), feed, rawFeed.etag, time.Now())
}

type parsedItem struct {
	url             string
	title           string
	publicationTime box.Time
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
		Title string `xml:"title"`
		Item  []Item `xml:"item"`
	}

	var rss struct {
		Channel Channel `xml:"channel"`
	}

	err := parseXML(body, &rss)
	if err != nil {
		return nil, err
	}

	var feed parsedFeed
	feed.name = rss.Channel.Title
	feed.items = make([]parsedItem, len(rss.Channel.Item))
	for i, item := range rss.Channel.Item {
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

	decoder.Entity = xml.HTMLEntity

	return decoder.Decode(doc)
}

// Try multiple time formats one after another until one works or all fail
func parseTime(value string) (bt box.Time, err error) {
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
			bt.Set(t)
			return bt, nil
		}
	}

	return bt, errors.New("Unable to parse time")
}
