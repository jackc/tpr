package data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Subscription struct {
	FeedID              pgtype.Text
	Name                pgtype.Text
	URL                 pgtype.Text
	LastFetchTime       pgtype.Timestamptz
	LastFailure         pgtype.Text
	LastFailureTime     pgtype.Timestamptz
	FailureCount        pgtype.Int4
	ItemCount           pgtype.Int8
	LastPublicationTime pgtype.Timestamptz
}

func InsertSubscription(ctx context.Context, db DB, userID string, feedURL string) error {
	return WithinTx(ctx, db, func(tx DB) error {
		_, err := tx.Exec(ctx, `insert into feeds(id, name, url, creation_time) values ($1,$2,$2,$3) on conflict (url) do nothing`, NewID(), feedURL, time.Now().UTC().Truncate(time.Microsecond))
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `insert into subscriptions(user_id, feed_id) select $1, id from feeds where url=$2`, userID, feedURL)
		return err
	})
}

const getSubscriptionsSQL = `select feeds.id as feed_id,
  name,
  feeds.url,
  last_fetch_time,
  last_failure,
  last_failure_time,
  failure_count,
  count(items.id) as item_count,
  max(items.publication_time::timestamptz) as last_publication_time
from feeds
  join subscriptions on feeds.id=subscriptions.feed_id
  left join items on feeds.id=items.feed_id
where user_id=$1
group by feeds.id, name, feeds.url, last_fetch_time, last_failure, last_failure_time, failure_count
order by name, feeds.id`

func SelectSubscriptions(ctx context.Context, db DB, userID string) ([]Subscription, error) {
	subs := make([]Subscription, 0, 16)
	rows, err := db.Query(ctx, getSubscriptionsSQL, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.FeedID, &s.Name, &s.URL, &s.LastFetchTime, &s.LastFailure, &s.LastFailureTime, &s.FailureCount, &s.ItemCount, &s.LastPublicationTime); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}

	return subs, rows.Err()
}

const deleteSubscriptionSQL = `delete from subscriptions where user_id=$1 and feed_id=$2`
const deleteFeedIfOrphanedSQL = `delete from feeds
where id=$1
  and not exists(select 1 from subscriptions where feed_id=id)`

func DeleteSubscription(ctx context.Context, db DB, userID, feedID string) error {
	return WithinTx(ctx, db, func(tx DB) error {
		if _, err := tx.Exec(ctx, deleteSubscriptionSQL, userID, feedID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, deleteFeedIfOrphanedSQL, feedID)
		return err
	})
}
