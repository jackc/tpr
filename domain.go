package main

import (
	"bytes"
	"code.google.com/p/go.crypto/scrypt"
	"crypto/rand"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/JackC/pgx"
	"io/ioutil"
	"net/http"
	"strconv"
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

	return repo.createUser(name, digest, salt)
}

func Subscribe(userID int32, feedURL string) (err error) {
	var conn *pgx.Connection
	conn, err = pool.Acquire()
	if err != nil {
		return err
	}
	defer pool.Release(conn)

	var feedID interface{}
	feedID, err = conn.SelectValue("select id from feeds where url=$1", feedURL)
	if _, ok := err.(pgx.NotSingleRowError); ok {
		var rawFeed *rawFeed
		rawFeed, err = fetchFeed(feedURL)
		if err != nil {
			return err
		}

		var feed *parsedFeed
		feed, err = parseFeed(rawFeed.body)
		if err != nil {
			return fmt.Errorf("Unable to parse feed: %v", err)
		}

		committed, txErr := conn.Transaction(func() bool {
			feedID, err = conn.SelectValue("insert into feeds(name, url, last_fetch_time, etag) values($1, $2, now(), $3) returning id", feed.name, feedURL, rawFeed.etag)
			if err != nil {
				return false
			}

			_, err = conn.Execute("insert into subscriptions(user_id, feed_id) values($1, $2)", userID, feedID)
			if err != nil {
				return false
			}

			insertSQL, insertArgs := buildNewItemsSQL(feedID.(int32), feed.items)
			_, err = conn.Execute(insertSQL, insertArgs...)
			if err != nil {
				return false
			}

			return true
		})
		if err != nil {
			return err
		}
		if txErr != nil {
			return err
		}
		if !committed {
			return errors.New("Commit failed")
		}

		return nil
	}
	if err != nil {
		return err
	}

	committed, txErr := conn.Transaction(func() bool {
		_, err = pool.Execute("insert into subscriptions(user_id, feed_id) values($1, $2)", userID, feedID)
		if err != nil {
			return false
		}

		_, err = pool.Execute(`
			insert into unread_items(user_id, feed_id, item_id)
			select $1, $2, item_id
			from items
			where items.user_id=$1
			  and items.feed_id=$2
		`, userID, feedID)
		if err != nil {
			return false
		}

		return true
	})
	if err != nil {
		return err
	}
	if txErr != nil {
		return err
	}
	if !committed {
		return errors.New("Commit failed")
	}

	return
}

func KeepFeedsFresh() {
	for {
		if staleFeeds, err := FindStaleFeeds(); err == nil {
			for _, sf := range staleFeeds {
				RefreshFeed(sf)
			}
		}
		time.Sleep(time.Minute)
	}
}

type staleFeed struct {
	id  int32
	url string
}

func FindStaleFeeds() (feeds []staleFeed, err error) {
	err = pool.SelectFunc("select id, url from feeds where last_fetch_time < now() - '10 minutes'::interval", func(r *pgx.DataRowReader) (err error) {
		var feed staleFeed
		feed.id = r.ReadValue().(int32)
		feed.url = r.ReadValue().(string)
		feeds = append(feeds, feed)
		return
	})

	return
}

type rawFeed struct {
	url  string
	body []byte
	etag string
}

func fetchFeed(url string) (feed *rawFeed, err error) {
	feed = &rawFeed{url: url}

	var resp *http.Response
	resp, err = http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Bad HTTP response: %s", resp.Status)
	}

	feed.body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Unable to read response body: %v", err)
	}

	feed.etag = resp.Header.Get("Etag")

	return feed, nil
}

func RefreshFeed(staleFeed staleFeed) {
	var conn *pgx.Connection
	var err error

	conn, err = pool.Acquire()
	if err != nil {
		return
	}
	defer pool.Release(conn)

	logFailure := func(failure string) {
		conn.Execute(`
			update feeds
			set last_failure=$1,
				last_failure_time=now(),
				failure_count=failure_count+1
			where id=$2`,
			failure, staleFeed.id)
	}

	var rawFeed *rawFeed
	rawFeed, err = fetchFeed(staleFeed.url)
	if err != nil {
		logFailure(err.Error())
		return
	}

	var feed *parsedFeed
	feed, err = parseFeed(rawFeed.body)
	if err != nil {
		logFailure(fmt.Sprintf("Unable to parse feed: %v", err))
		return
	}

	conn.Transaction(func() bool {
		_, err = conn.Execute(`
			update feeds
			set name=$1,
		 	  last_fetch_time=now(),
		 	  etag=$2,
		 	  last_failure=null,
		 	  last_failure_time=null,
		 	  failure_count=0
			where id=$3`,
			feed.name,
			rawFeed.etag,
			staleFeed.id)
		if err != nil {
			return false
		}

		insertSQL, insertArgs := buildNewItemsSQL(staleFeed.id, feed.items)
		_, err = conn.Execute(insertSQL, insertArgs...)
		if err != nil {
			return false
		}

		return true
	})
}

func buildNewItemsSQL(feedID int32, items []parsedItem) (sql string, args []interface{}) {
	var buf bytes.Buffer
	args = append(args, feedID)

	buf.WriteString(`
			with new_items as (
				insert into items(feed_id, url, title, publication_time)
				select $1, url, title, publication_time
				from (values
		`)

	for i, item := range items {
		if i > 0 {
			buf.WriteString(",")
		}

		buf.WriteString("($")

		args = append(args, item.url)
		buf.WriteString(strconv.FormatInt(int64(len(args)), 10))

		buf.WriteString(",$")

		args = append(args, item.title)
		buf.WriteString(strconv.FormatInt(int64(len(args)), 10))

		buf.WriteString(",$")

		args = append(args, item.publicationTime)
		buf.WriteString(strconv.FormatInt(int64(len(args)), 10))

		buf.WriteString("::timestamptz)")
	}

	buf.WriteString(`
			) t(url, title, publication_time)
			where not exists(
				select 1
				from items
				where digest=digest_item($1, t.publication_time, t.title, t.url)
			)
			returning id
		)
		insert into unread_items(user_id, feed_id, item_id)
		select user_id, $1, new_items.id
		from subscriptions
		  cross join new_items
		where subscriptions.feed_id=$1
	`)

	return buf.String(), args
}

// func fetchFeed(url string, etag string) (body string, err error) {
// 	var req *http.Request
// 	var resp *http.Response

// 	req, err = http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return err
// 	}

// 	if etag != "" {
// 		req.Header.Add("If-None-Match", "etag")
// 	}

// 	resp, err = client.Do(req)
// 	defer resp.Body.Close()
// 	if err != nil {
// 		return err
// 	}

// 	return
// }

// type fetchedFeed struct {
//         name string
//       url string
//       time time.Time
//       etag string
//       last_failure varchar,
//       last_failure_time timestamp with time zone,
// }

type parsedItem struct {
	url             string
	title           string
	publicationTime time.Time
}

func (i *parsedItem) isValid() bool {
	var zeroTime time.Time

	return i.url != "" && i.title != "" && i.publicationTime != zeroTime
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

	err := xml.Unmarshal(body, &rss)
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

	err := xml.Unmarshal(body, &atom)
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

// Try multiple time formats one after another until one works or all fail
func parseTime(value string) (t time.Time, err error) {
	formats := []string{
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		time.RFC822,
		"02 Jan 2006 15:04 MST",    // RFC822 with 4 digit year
		"02 Jan 2006 15:04:05 MST", // RFC822 with 4 digit year and seconds
		time.RFC1123,
		time.RFC1123Z,
	}
	for _, f := range formats {
		t, err = time.Parse(f, value)
		if err == nil {
			return
		}
	}

	return t, errors.New("Unable to parse time")
}
