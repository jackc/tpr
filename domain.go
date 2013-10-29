package main

import (
	"code.google.com/p/go.crypto/scrypt"
	"crypto/rand"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/JackC/pgx"
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

	var v interface{}
	v, err = pool.SelectValue("insert into users(name, password_digest, password_salt) values($1, $2, $3) returning id", name, digest, salt)
	if err != nil {
		return
	}
	userID = v.(int32)

	return
}

func Subscribe(userID int32, feedURL string) (err error) {
	var feedID interface{}
	feedID, err = pool.SelectValue("select id from feeds where url=$1", feedURL)
	if _, ok := err.(pgx.NotSingleRowError); ok {
		var resp *http.Response
		resp, err = http.Get(feedURL)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("Bad HTTP response: %s", resp.Status)
		}
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Unable to read response body: %v", err)
		}

		var feed *parsedFeed
		feed, err = parseRSS(body)
		if err != nil {
			return fmt.Errorf("Unable to parse feed: %v", err)
		}

		var conn *pgx.Connection
		conn, err = pool.Acquire()
		if err != nil {
			return err
		}
		defer pool.Release(conn)

		committed, txErr := conn.Transaction(func() bool {
			feedID, err = conn.SelectValue("insert into feeds(name, url, last_fetch_time, etag) values($1, $2, now(), $3) returning id", feed.name, feedURL, resp.Header.Get("Etag"))
			if err != nil {
				return false
			}

			for _, item := range feed.items {
				_, err = conn.Execute("insert into items(feed_id, url, title, body, publication_time) values($1, $2, $3, $4, $5)", feedID, item.url, item.title, item.body, item.publicationTime)
				if err != nil {
					return false
				}
			}

			_, err = conn.Execute("insert into subscriptions(user_id, feed_id) values($1, $2)", userID, feedID)
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

	_, err = pool.Execute("insert into subscriptions(user_id, feed_id) values($1, $2)", userID, feedID)
	if err != nil {
		return err
	}
	return
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
	body            string
	publicationTime time.Time
}

type parsedFeed struct {
	name  string
	items []parsedItem
}

func parseRSS(body []byte) (*parsedFeed, error) {
	type Item struct {
		Link        string `xml:"link"`
		Title       string `xml:"title"`
		Description string `xml:"description"`
		Date        string `xml:"date"`
	}

	type Channel struct {
		Title string `xml:"title"`
		Item  []Item `xml:"item"`
	}

	var rss struct {
		Channel Channel `xml:"channel"`
	}

	err := xml.Unmarshal(body, &rss)

	var feed parsedFeed
	feed.name = rss.Channel.Title
	feed.items = make([]parsedItem, len(rss.Channel.Item))
	for i, item := range rss.Channel.Item {
		feed.items[i].url = item.Link
		feed.items[i].title = item.Title
		feed.items[i].body = item.Description
		feed.items[i].publicationTime, _ = time.Parse("2006-01-02T15:04:05-07:00", item.Date)
	}

	return &feed, err
}
