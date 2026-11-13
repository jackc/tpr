package data

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgxutil"
)

const markItemReadSQL = `delete from unread_items
where user_id=$1
  and item_id=$2`

func MarkItemRead(ctx context.Context, db DB, userID, itemID string) error {
	_, err := pgxutil.ExecRow(ctx, db, markItemReadSQL, userID, itemID)
	return err
}

const getFeedsForUserSQL = `select feeds.id::text as feed_id,
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
  group by feeds.id, name, feeds.url, last_fetch_time, last_failure, last_failure_time, failure_count
  order by name, feeds.id`

func CopySubscriptionsForUserAsJSON(ctx context.Context, db DB, w io.Writer, userID string) error {
	return copyJSON(ctx, db, w, getFeedsForUserSQL, userID)
}

const getUnreadItemsSQL = `select
    items.id::text as id,
    feeds.id::text as feed_id,
    feeds.name as feed_name,
    items.title,
    items.url,
    extract(epoch from coalesce(publication_time, items.creation_time)::timestamptz(0)) as publication_time
  from feeds
    join items on feeds.id=items.feed_id
    join unread_items on items.id=unread_items.item_id
  where user_id=$1
  order by publication_time asc, items.id`

func CopyUnreadItemsAsJSONByUserID(ctx context.Context, db DB, w io.Writer, userID string) error {
	return copyJSON(ctx, db, w, getUnreadItemsSQL, userID)
}

const getArchivedItemsSQL = `select
    items.id::text as id,
    feeds.id::text as feed_id,
    feeds.name as feed_name,
    items.title,
    items.url,
    extract(epoch from coalesce(publication_time, items.creation_time)::timestamptz(0)) as publication_time
  from feeds
    join subscriptions on feeds.id=subscriptions.feed_id
    join items on feeds.id=items.feed_id
  where user_id=$1
  order by publication_time desc, items.id
  limit $2`

func CopyArchivedItemsAsJSONByUserID(ctx context.Context, db DB, w io.Writer, userID string) error {
	return copyJSON(ctx, db, w, getArchivedItemsSQL, userID, 250)
}

type ParsedItem struct {
	URL             string
	Title           string
	PublicationTime pgtype.Timestamptz
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

func UpdateFeedWithFetchSuccess(ctx context.Context, db DB, feedID string, update *ParsedFeed, etag pgtype.Text, fetchTime time.Time) error {
	return WithinTx(ctx, db, func(tx DB) error {
		_, err := tx.Exec(ctx, updateFeedWithFetchSuccessSQL, update.Name, fetchTime, &etag, feedID)
		if err != nil {
			return err
		}
		for _, item := range update.Items {
			id := NewID()
			tag, err := tx.Exec(ctx, `insert into items(id, feed_id, url, title, publication_time, creation_time)
    values ($1,$2,$3,$4,$5,$6) on conflict (feed_id,url) do nothing`, id, feedID, item.URL, item.Title, item.PublicationTime, fetchTime.UTC().Truncate(time.Microsecond))
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				continue
			}
			_, err = tx.Exec(ctx, `insert into unread_items(user_id,feed_id,item_id) select user_id,feed_id,$1 from subscriptions where feed_id=$2`, id, feedID)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

const updateFeedWithFetchUnchangedSQL = `update feeds
set last_fetch_time=$1,
  last_failure=null,
  last_failure_time=null,
  failure_count=0
where id=$2`

func UpdateFeedWithFetchUnchanged(ctx context.Context, db DB, feedID string, fetchTime time.Time) (err error) {
	_, err = db.Exec(ctx, updateFeedWithFetchUnchangedSQL, fetchTime, feedID)
	return
}

const updateFeedWithFetchFailureSQL = `update feeds
set last_failure=$1,
  last_failure_time=$2,
  failure_count=failure_count+1
where id=$3`

func UpdateFeedWithFetchFailure(ctx context.Context, db DB, feedID string, failure string, fetchTime time.Time) (err error) {
	_, err = db.Exec(ctx, updateFeedWithFetchFailureSQL, failure, fetchTime, feedID)
	return err
}

const getFeedsUncheckedSinceSQL = `select id, url, etag
from feeds
where greatest(last_fetch_time, last_failure_time, '-Infinity'::timestamptz) < $1 order by id`

func GetFeedsUncheckedSince(ctx context.Context, db DB, since time.Time) ([]Feed, error) {
	feeds := make([]Feed, 0, 8)
	rows, err := db.Query(ctx, getFeedsUncheckedSinceSQL, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var feed Feed
		if err := rows.Scan(&feed.ID, &feed.URL, &feed.ETag); err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}

	return feeds, rows.Err()
}

func copyJSON(ctx context.Context, db DB, w io.Writer, sql string, args ...any) error {
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return err
	}
	if records == nil {
		records = []map[string]any{}
	}
	return json.NewEncoder(w).Encode(records)
}
