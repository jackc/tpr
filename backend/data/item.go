package data

import (
	"bytes"
	"io"
	"strconv"
	"time"

	"github.com/jackc/pgx"
)

const markItemReadSQL = `delete from unread_items
where user_id=$1
  and item_id=$2`

func MarkItemRead(db Queryer, userID, itemID int32) error {
	commandTag, err := prepareExec(db, "markItemRead", markItemReadSQL, userID, itemID)
	if err != nil {
		return err
	}
	if commandTag != "DELETE 1" {
		return ErrNotFound
	}

	return nil
}

const getFeedsForUserSQL = `select coalesce(json_agg(row_to_json(t)), '[]'::json)
from (
  select feeds.id as feed_id,
    name,
    feeds.url,
    extract(epoch from last_fetch_time::timestamptz(0)) as last_fetch_time,
    last_failure,
    extract(epoch from last_failure_time::timestamptz(0)) as last_failure_time,
    failure_count,
    count(items.id) as item_count,
    extract(epoch from max(items.publication_time::timestamptz(0))) as last_publication_time
  from feeds
    join subscriptions on feeds.id=subscriptions.feed_id
    left join items on feeds.id=items.feed_id
  where user_id=$1
  group by feeds.id
  order by name
) t`

func CopySubscriptionsForUserAsJSON(db Queryer, w io.Writer, userID int32) error {
	var b []byte
	err := prepareQueryRow(db, "getFeedsForUser", getFeedsForUserSQL, userID).Scan(&b)
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

const getUnreadItemsSQL = `select coalesce(json_agg(row_to_json(t)), '[]'::json)
from (
  select
    items.id,
    feeds.id as feed_id,
    feeds.name as feed_name,
    items.title,
    items.url,
    extract(epoch from coalesce(publication_time, items.creation_time)::timestamptz(0)) as publication_time
  from feeds
    join items on feeds.id=items.feed_id
    join unread_items on items.id=unread_items.item_id
  where user_id=$1
  order by publication_time asc
) t`

func CopyUnreadItemsAsJSONByUserID(db Queryer, w io.Writer, userID int32) error {
	var b []byte
	err := prepareQueryRow(db, "getUnreadItems", getUnreadItemsSQL, userID).Scan(&b)
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

type ParsedItem struct {
	URL             string
	Title           string
	PublicationTime Time
}

func (i *ParsedItem) IsValid() bool {
	return i.URL != "" && i.Title != ""
}

type ParsedFeed struct {
	Name  string
	Items []ParsedItem
}

func (f *ParsedFeed) IsValid() bool {
	if f.Name == "" {
		return false
	}

	for _, item := range f.Items {
		if !item.IsValid() {
			return false
		}
	}

	return true
}

const updateFeedWithFetchSuccessSQL = `
      update feeds
      set name=$1,
        last_fetch_time=$2,
        etag=$3,
        last_failure=null,
        last_failure_time=null,
        failure_count=0
      where id=$4`

func UpdateFeedWithFetchSuccess(db *pgx.ConnPool, feedID int32, update *ParsedFeed, etag String, fetchTime time.Time) error {
	_, err := db.Prepare("updateFeedWithFetchSuccess", updateFeedWithFetchSuccessSQL)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("updateFeedWithFetchSuccess",
		update.Name,
		fetchTime,
		&etag,
		feedID)
	if err != nil {
		return err
	}

	if len(update.Items) > 0 {
		insertSQL, insertArgs := buildNewItemsSQL(feedID, update.Items)
		_, err = tx.Exec(insertSQL, insertArgs...)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

const updateFeedWithFetchUnchangedSQL = `update feeds
set last_fetch_time=$1,
  last_failure=null,
  last_failure_time=null,
  failure_count=0
where id=$2`

func UpdateFeedWithFetchUnchanged(db Queryer, feedID int32, fetchTime time.Time) (err error) {
	_, err = prepareExec(db, "updateFeedWithFetchUnchanged", updateFeedWithFetchUnchangedSQL, fetchTime, feedID)
	return
}

const updateFeedWithFetchFailureSQL = `update feeds
set last_failure=$1,
  last_failure_time=$2,
  failure_count=failure_count+1
where id=$3`

func UpdateFeedWithFetchFailure(db Queryer, feedID int32, failure string, fetchTime time.Time) (err error) {
	_, err = prepareExec(db, "updateFeedWithFetchFailure", updateFeedWithFetchFailureSQL, failure, fetchTime, feedID)
	return err
}

func buildNewItemsSQL(feedID int32, items []ParsedItem) (sql string, args []interface{}) {
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
		args = append(args, item.URL)
		buf.WriteString(strconv.FormatInt(int64(len(args)), 10))

		buf.WriteString(",$")
		args = append(args, item.Title)
		buf.WriteString(strconv.FormatInt(int64(len(args)), 10))

		buf.WriteString(",$")
		if item.PublicationTime.Status == Present {
			args = append(args, item.PublicationTime.Value)
		} else {
			args = append(args, nil)
		}
		buf.WriteString(strconv.FormatInt(int64(len(args)), 10))
		buf.WriteString("::timestamptz)")
	}

	buf.WriteString(`
      ) t(url, title, publication_time)
      where not exists(
        select 1
        from items
        where feed_id=$1
          and url=t.url
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

const getFeedsUncheckedSinceSQL = `select id, url, etag
from feeds
where greatest(last_fetch_time, last_failure_time, '-Infinity'::timestamptz) < $1`

func GetFeedsUncheckedSince(db Queryer, since time.Time) ([]Feed, error) {
	feeds := make([]Feed, 0, 8)
	rows, _ := prepareQuery(db, "getFeedsUncheckedSince", getFeedsUncheckedSinceSQL, since)

	for rows.Next() {
		var feed Feed
		rows.Scan(&feed.ID, &feed.URL, &feed.ETag)
		feeds = append(feeds, feed)
	}

	return feeds, rows.Err()
}
