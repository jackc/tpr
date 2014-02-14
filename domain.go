package main

import (
	"bytes"
	"code.google.com/p/go.crypto/scrypt"
	"crypto/rand"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/JackC/box"
	"io/ioutil"
	"net/http"
	"time"
)

var client *http.Client

func init() {
	client = &http.Client{}
}

func CreateUser(name string, password string) (userID int32, err error) {
	salt := make([]byte, 8)
	_, _ = rand.Read(salt)

	var digest []byte
	digest, err = scrypt.Key([]byte(password), salt, 16384, 8, 1, 32)
	if err != nil {
		return
	}

	return repo.CreateUser(name, digest, salt)
}

func Subscribe(userID int32, feedURL string) (err error) {
	return repo.CreateSubscription(userID, feedURL)
}

func KeepFeedsFresh() {
	for {
		t := time.Now().Add(-10 * time.Minute)
		if staleFeeds, err := repo.GetFeedsUncheckedSince(t); err == nil {
			logger.Info("tpr", fmt.Sprintf("Found %d stale feeds", len(staleFeeds)))
			for _, sf := range staleFeeds {
				RefreshFeed(sf)
			}
		} else {
			logger.Error("tpr", fmt.Sprintf("repo.GetFeedsUncheckedSince failed: %v", err))
		}
		time.Sleep(time.Minute)
	}
}

type rawFeed struct {
	url  string
	body []byte
	etag string
}

func fetchFeed(url, etag string) (feed *rawFeed, err error) {
	client := &http.Client{}

	feed = &rawFeed{url: url}

	req, err := http.NewRequest("GET", feed.url, nil)
	if etag != "" {
		req.Header.Add("If-None-Match", etag)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		feed.body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("Unable to read response body: %v", err)
		}

		feed.etag = resp.Header.Get("Etag")

		return feed, nil
	case 304:
		return nil, nil
	default:
		return nil, fmt.Errorf("Bad HTTP response: %s", resp.Status)
	}
}

func RefreshFeed(staleFeed staleFeed) {
	rawFeed, err := fetchFeed(staleFeed.url, staleFeed.etag)
	if err != nil {
		logger.Error("tpr", fmt.Sprintf("fetchFeed %s failed: %v", staleFeed.url, err))
		repo.UpdateFeedWithFetchFailure(staleFeed.id, err.Error(), time.Now())
		return
	}
	// 304 unchanged
	if rawFeed == nil {
		logger.Info("tpr", fmt.Sprintf("fetchFeed %s 304 unchanged", staleFeed.url))
		repo.UpdateFeedWithFetchUnchanged(staleFeed.id, time.Now())
		return
	}

	feed, err := parseFeed(rawFeed.body)
	if err != nil {
		logger.Error("tpr", fmt.Sprintf("parseFeed %s failed: %v", staleFeed.url, err))
		repo.UpdateFeedWithFetchFailure(staleFeed.id, fmt.Sprintf("Unable to parse feed: %v", err), time.Now())
		return
	}

	logger.Info("tpr", fmt.Sprintf("refreshFeed %s (%d) succeeded", staleFeed.url, staleFeed.id))
	repo.UpdateFeedWithFetchSuccess(staleFeed.id, feed, rawFeed.etag, time.Now())
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
