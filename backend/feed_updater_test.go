package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/tpr/backend/data"
	log "gopkg.in/inconshreveable/log15.v2"
)

var feedParsingTests = []struct {
	name       string
	body       []byte
	parsedFeed *data.ParsedFeed
	errMsg     string
}{
	{"RSS - Minimal",
		[]byte(`<?xml version='1.0' encoding='UTF-8'?>
<rss>
  <channel>
    <title>News</title>
    <item>
      <title>Snow Storm</title>
      <link>http://example.org/snow-storm</link>
      <pubDate>Fri, 03 Jan 2014 22:45:00 GMT</pubDate>
    </item>
    <item>
      <title>Blizzard</title>
      <link>http://example.org/blizzard</link>
      <pubDate>Sat, 04 Jan 2014 08:15:00 GMT</pubDate>
    </item>
  </channel>
</rss>
</xml>`),
		&data.ParsedFeed{
			Name: "News",
			Items: []data.ParsedItem{
				{
					Title:           "Snow Storm",
					URL:             "http://example.org/snow-storm",
					PublicationTime: pgtype.Timestamptz{Time: time.Date(2014, 1, 3, 22, 45, 0, 0, time.UTC), Status: pgtype.Present},
				},
				{
					Title:           "Blizzard",
					URL:             "http://example.org/blizzard",
					PublicationTime: pgtype.Timestamptz{Time: time.Date(2014, 1, 4, 8, 15, 0, 0, time.UTC), Status: pgtype.Present},
				},
			}},
		"",
	},
	{"RSS - v1",
		[]byte(`<?xml version='1.0' encoding='UTF-8'?>
<rdf>
  <channel>
    <title>News</title>
  </channel>
  <item>
    <title>Snow Storm</title>
    <link>http://example.org/snow-storm</link>
    <date>Fri, 03 Jan 2014 22:45:00 GMT</date>
  </item>
  <item>
    <title>Blizzard</title>
    <link>http://example.org/blizzard</link>
    <date>Sat, 04 Jan 2014 08:15:00 GMT</date>
  </item>
</rdf>
</xml>`),
		&data.ParsedFeed{
			Name: "News",
			Items: []data.ParsedItem{
				{
					Title:           "Snow Storm",
					URL:             "http://example.org/snow-storm",
					PublicationTime: pgtype.Timestamptz{Time: time.Date(2014, 1, 3, 22, 45, 0, 0, time.UTC), Status: pgtype.Present},
				},
				{
					Title:           "Blizzard",
					URL:             "http://example.org/blizzard",
					PublicationTime: pgtype.Timestamptz{Time: time.Date(2014, 1, 4, 8, 15, 0, 0, time.UTC), Status: pgtype.Present},
				},
			}},
		"",
	},
	{"RSS - Valid entities converted to UTF-8",
		[]byte(`<?xml version='1.0' encoding='UTF-8'?>
<rss>
  <channel>
    <title>Joe&#160;Blogger&#039;s Site</title>
    <item>
      <title>Snow Storm</title>
      <link>http://example.org/snow-storm</link>
      <pubDate>Fri, 03 Jan 2014 22:45:00 GMT</pubDate>
    </item>
  </channel>
</rss>
</xml>`),
		&data.ParsedFeed{
			Name: "Joe\u00a0Blogger's Site",
			Items: []data.ParsedItem{
				{
					Title:           "Snow Storm",
					URL:             "http://example.org/snow-storm",
					PublicationTime: pgtype.Timestamptz{Time: time.Date(2014, 1, 3, 22, 45, 0, 0, time.UTC), Status: pgtype.Present},
				},
			}},
		"",
	},
	{"RSS - Invalid entities...",
		[]byte(`<?xml version='1.0' encoding='UTF-8'?>
<rss>
  <channel>
    <title>Joe&nbsp;Blogger</title>
    <item>
      <title>Snow Storm</title>
      <link>http://example.org/snow-storm</link>
      <pubDate>Fri, 03 Jan 2014 22:45:00 GMT</pubDate>
    </item>
  </channel>
</rss>
</xml>`),
		&data.ParsedFeed{
			Name: "Joe\u00a0Blogger",
			Items: []data.ParsedItem{
				{
					Title:           "Snow Storm",
					URL:             "http://example.org/snow-storm",
					PublicationTime: pgtype.Timestamptz{Time: time.Date(2014, 1, 3, 22, 45, 0, 0, time.UTC), Status: pgtype.Present},
				},
			}},
		"",
	},
	{"RSS - Without item dates",
		[]byte(`<?xml version='1.0' encoding='UTF-8'?>
<rss>
  <channel>
    <title>News</title>
    <item>
      <title>Snow Storm</title>
      <link>http://example.org/snow-storm</link>
    </item>
  </channel>
</rss>
</xml>`),
		&data.ParsedFeed{
			Name: "News",
			Items: []data.ParsedItem{
				{
					Title: "Snow Storm",
					URL:   "http://example.org/snow-storm",
				},
			}},
		"",
	},
	{"RSS - Empty Channel Title",
		[]byte(`<?xml version="1.0" encoding="utf-8" ?>
<rss>
  <channel>
    <title></title>
    <description>Description instead of title</description>
    <item>
      <title>Snow Storm</title>
      <link>http://example.org/snow-storm</link>
    </item>
  </channel>
</rss>
`),
		&data.ParsedFeed{
			Name: "Description instead of title",
			Items: []data.ParsedItem{
				{
					Title: "Snow Storm",
					URL:   "http://example.org/snow-storm",
				},
			}},
		"",
	},

	{"Atom - Minimal",
		[]byte(`<?xml version='1.0' encoding='UTF-8'?>
<feed>
  <title>News</title>
  <entry>
    <title>Snow Storm</title>
    <link href="http://example.org/snow-storm" />
    <published>2014-01-03T22:45:00Z</published>
  </entry>
  <entry>
    <title>Blizzard</title>
    <link href="http://example.org/blizzard" />
    <published>2014-01-04T08:15:00Z</published>
  </entry>
</feed>
</xml>`),
		&data.ParsedFeed{
			Name: "News",
			Items: []data.ParsedItem{
				{
					Title:           "Snow Storm",
					URL:             "http://example.org/snow-storm",
					PublicationTime: pgtype.Timestamptz{Time: time.Date(2014, 1, 3, 22, 45, 0, 0, time.UTC), Status: pgtype.Present},
				},
				{
					Title:           "Blizzard",
					URL:             "http://example.org/blizzard",
					PublicationTime: pgtype.Timestamptz{Time: time.Date(2014, 1, 4, 8, 15, 0, 0, time.UTC), Status: pgtype.Present},
				},
			}},
		"",
	},
	{"Atom - with encoding ISO-8859-1",
		[]byte(`<?xml version='1.0' encoding='ISO-8859-1'?>
<feed>
  <title>News</title>
  <entry>
    <title>Snow Storm</title>
    <link href="http://example.org/snow-storm" />
    <published>2014-01-03T22:45:00Z</published>
  </entry>
  <entry>
    <title>Blizzard</title>
    <link href="http://example.org/blizzard" />
    <published>2014-01-04T08:15:00Z</published>
  </entry>
</feed>
</xml>`),
		&data.ParsedFeed{
			Name: "News",
			Items: []data.ParsedItem{
				{
					Title:           "Snow Storm",
					URL:             "http://example.org/snow-storm",
					PublicationTime: pgtype.Timestamptz{Time: time.Date(2014, 1, 3, 22, 45, 0, 0, time.UTC), Status: pgtype.Present},
				},
				{
					Title:           "Blizzard",
					URL:             "http://example.org/blizzard",
					PublicationTime: pgtype.Timestamptz{Time: time.Date(2014, 1, 4, 8, 15, 0, 0, time.UTC), Status: pgtype.Present},
				},
			}},
		"",
	},
}

func TestParseFeed(t *testing.T) {
	for i, tt := range feedParsingTests {
		actual, err := parseFeed(tt.body)
		if err != nil && err.Error() != tt.errMsg {
			t.Errorf("%d. %s: Unexpected error: %v", i, tt.name, err)
		}
		if actual == nil {
			if tt.parsedFeed != nil {
				t.Errorf("%d. %s: Actual parsed feed should not have been nil, but it was", i, tt.name)
			}
			continue
		}
		if tt.parsedFeed == nil {
			t.Errorf("%d. %s: Actual parsed feed should have been nil, but it was not", i, tt.name)
			continue
		}
		if actual.Name != tt.parsedFeed.Name {
			t.Errorf("%d. %s: Expected name to be %#v, but it was %#v", i, tt.name, tt.parsedFeed.Name, actual.Name)
		}
		if len(actual.Items) != len(tt.parsedFeed.Items) {
			t.Errorf("%d. %s: Expected %d items, but instead found %d items", i, tt.name, len(tt.parsedFeed.Items), len(actual.Items))
			continue
		}
		for j, actualItem := range actual.Items {
			expectedItem := tt.parsedFeed.Items[j]
			if actualItem.Title != expectedItem.Title {
				t.Errorf("%d. %s Item %d: Expected title %#v, but is was %#v", i, tt.name, j, expectedItem.Title, actualItem.Title)
			}
			if actualItem.URL != expectedItem.URL {
				t.Errorf("%d. %s Item %d: Expected url %#v, but is was %#v", i, tt.name, j, expectedItem.URL, actualItem.URL)
			}
			if actualItem.PublicationTime.Status == expectedItem.PublicationTime.Status {
				if actualItem.PublicationTime.Status == pgtype.Present && !actualItem.PublicationTime.Time.Equal(expectedItem.PublicationTime.Time) {
					t.Errorf("%d. %s Item %d: Expected publicationTime %v, but is was %v", i, tt.name, j, expectedItem.PublicationTime, actualItem.PublicationTime)
				}
			} else {
				t.Errorf("%d. %s Item %d: Expected publicationTime status %v, but is was %v", i, tt.name, j, expectedItem.PublicationTime.Status, actualItem.PublicationTime.Status)
			}
		}
	}
}

var timeParsingTests = []struct {
	unparsed string
	expected time.Time
	errMsg   string
}{
	{"2010-07-13T14:15:32-07:00", time.Date(2010, 7, 13, 21, 15, 32, 0, time.UTC), ""},
	{"2010-07-13T14:15:32Z", time.Date(2010, 7, 13, 14, 15, 32, 0, time.UTC), ""},
	{"Fri, 03 Jan 2014 22:45:00 GMT", time.Date(2014, 1, 3, 22, 45, 0, 0, time.UTC), ""},
	{"03 Jan 2014 22:45 GMT", time.Date(2014, 1, 3, 22, 45, 0, 0, time.UTC), ""},
	{"03 Jan 2014 22:45 GMT", time.Date(2014, 1, 3, 22, 45, 0, 0, time.UTC), ""},
	{"Fri, 3 Jan 2014 16:35:05 -0800", time.Date(2014, 1, 4, 0, 35, 5, 0, time.UTC), ""},
	{"Sat, 04 Jan 2014", time.Date(2014, 1, 4, 0, 0, 0, 0, time.UTC), ""},
	{"2011-05-19", time.Date(2011, 5, 19, 0, 0, 0, 0, time.UTC), ""},
}

func TestParseTime(t *testing.T) {
	for i, tt := range timeParsingTests {
		actual, err := parseTime(tt.unparsed)
		if err != nil && err.Error() != tt.errMsg {
			t.Errorf("%d. %s: Unexpected error: %v", i, tt.unparsed, err)
			continue
		}
		if !tt.expected.Equal(actual.Time) {
			t.Errorf("%d. %s: expected to parse to %v, but instead was %v", i, tt.unparsed, tt.expected, actual)
		}
	}
}

func TestFetchFeed(t *testing.T) {
	pool := newConnPool(t)

	rssBody := []byte(`<?xml version='1.0' encoding='UTF-8'?>
<rss>
  <channel>
    <title>News</title>
    <item>
      <title>Snow Storm</title>
      <link>http://example.org/snow-storm</link>
      <pubDate>Fri, 03 Jan 2014 22:45:00 GMT</pubDate>
    </item>
    <item>
      <title>Blizzard</title>
      <link>http://example.org/blizzard</link>
      <pubDate>Sat, 04 Jan 2014 08:15:00 GMT</pubDate>
    </item>
  </channel>
</rss>
</xml>`)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(rssBody)
	}))
	defer ts.Close()

	u := NewFeedUpdater(pool, log.Root())
	rawFeed, err := u.fetchFeed(ts.URL, pgtype.Varchar{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if rawFeed.url != ts.URL {
		t.Errorf("rawFeed.url should match requested url but instead it was: %v", rawFeed.url)
	}
	if bytes.Compare(rssBody, rawFeed.body) != 0 {
		t.Errorf("rawFeed body should match returned body but instead it was: %v", rawFeed.body)
	}
	if rawFeed.etag.Status != pgtype.Null {
		t.Errorf("Expected no ETag to be null but instead it was: %v", rawFeed.etag)
	}
}
