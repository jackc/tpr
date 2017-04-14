package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
	"github.com/jackc/tpr/backend/data"
	"golang.org/x/net/html/charset"
	log "gopkg.in/inconshreveable/log15.v2"
)

type FeedUpdater struct {
	client                   *http.Client
	maxConcurrentFeedFetches int
	pool                     *pgx.ConnPool
	logger                   log.Logger
}

func NewFeedUpdater(pool *pgx.ConnPool, logger log.Logger) *FeedUpdater {
	feedUpdater := &FeedUpdater{}
	feedUpdater.pool = pool
	feedUpdater.logger = logger
	feedUpdater.client = &http.Client{Timeout: 60 * time.Second}
	feedUpdater.maxConcurrentFeedFetches = 25
	return feedUpdater
}

func (u *FeedUpdater) KeepFeedsFresh() {

	for {
		startTime := time.Now()

		if staleFeeds, err := data.GetFeedsUncheckedSince(u.pool, startTime.Add(-10*time.Minute)); err == nil {
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
	etag pgtype.Varchar
}

func (u *FeedUpdater) fetchFeed(feedURL string, etag pgtype.Varchar) (*rawFeed, error) {
	feed := &rawFeed{url: feedURL}

	req, err := http.NewRequest("GET", feed.url, nil)
	if etag.Status == pgtype.Present {
		req.Header.Add("If-None-Match", etag.String)
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

		feed.etag = newStringFallback(resp.Header.Get("Etag"), pgtype.Null)

		return feed, nil
	case 304:
		return nil, nil
	default:
		return nil, fmt.Errorf("Bad HTTP response: %s", resp.Status)
	}
}

func (u *FeedUpdater) RefreshFeed(staleFeed data.Feed) {
	rawFeed, err := u.fetchFeed(staleFeed.URL.String, staleFeed.ETag)
	if err != nil {
		u.logger.Error("fetchFeed failed", "url", staleFeed.URL.String, "error", err)
		data.UpdateFeedWithFetchFailure(u.pool, staleFeed.ID.Int, err.Error(), time.Now())
		return
	}
	// 304 unchanged
	if rawFeed == nil {
		u.logger.Info("fetchFeed 304 unchanged", "url", staleFeed.URL.Value)
		data.UpdateFeedWithFetchUnchanged(u.pool, staleFeed.ID.Int, time.Now())
		return
	}

	feed, err := parseFeed(rawFeed.body)
	if err != nil {
		u.logger.Error("parseFeed failed", "url", staleFeed.URL.Value, "error", err)
		data.UpdateFeedWithFetchFailure(u.pool, staleFeed.ID.Int, fmt.Sprintf("Unable to parse feed: %v", err), time.Now())
		return
	}

	u.logger.Info("refreshFeed succeeded", "url", staleFeed.URL.Value, "id", staleFeed.ID.Int)
	data.UpdateFeedWithFetchSuccess(u.pool, staleFeed.ID.Int, feed, rawFeed.etag, time.Now())
}

func parseFeed(body []byte) (f *data.ParsedFeed, err error) {
	f, err = parseRSS(body)
	if err == nil {
		return f, nil
	}

	return parseAtom(body)
}

func parseRSS(body []byte) (*data.ParsedFeed, error) {
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

	var feed data.ParsedFeed
	if rss.Channel.Title != "" {
		feed.Name = rss.Channel.Title
	} else {
		feed.Name = rss.Channel.Description
	}

	var items []Item
	if len(rss.Item) > 0 {
		items = rss.Item
	} else {
		items = rss.Channel.Item
	}

	feed.Items = make([]data.ParsedItem, len(items))
	for i, item := range items {
		feed.Items[i].URL = item.Link
		feed.Items[i].Title = item.Title
		if item.Date != "" {
			feed.Items[i].PublicationTime, _ = parseTime(item.Date)
		}
		if item.PubDate != "" {
			feed.Items[i].PublicationTime, _ = parseTime(item.PubDate)
		}
	}

	if !feed.IsValid() {
		return nil, errors.New("Invalid RSS")
	}

	return &feed, nil
}

func parseAtom(body []byte) (*data.ParsedFeed, error) {
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

	var feed data.ParsedFeed
	feed.Name = atom.Title
	feed.Items = make([]data.ParsedItem, len(atom.Entry))
	for i, entry := range atom.Entry {
		feed.Items[i].URL = entry.Link.Href
		feed.Items[i].Title = entry.Title
		if entry.Published != "" {
			feed.Items[i].PublicationTime, _ = parseTime(entry.Published)
		}
		if entry.Updated != "" {
			feed.Items[i].PublicationTime, _ = parseTime(entry.Updated)
		}
	}

	if !feed.IsValid() {
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
func parseTime(value string) (pgtype.Timestamptz, error) {
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
			return pgtype.Timestamptz{Time: t, Status: pgtype.Present}, nil
		}
	}

	return pgtype.Timestamptz{}, errors.New("Unable to parse time")
}
