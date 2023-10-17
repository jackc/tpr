package data

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgxutil"
)

type Subscription struct {
	FeedID              pgtype.Int4
	Name                pgtype.Text
	URL                 pgtype.Text
	LastFetchTime       pgtype.Timestamptz
	LastFailure         pgtype.Text
	LastFailureTime     pgtype.Timestamptz
	FailureCount        pgtype.Int4
	ItemCount           pgtype.Int8
	LastPublicationTime pgtype.Timestamptz
}

const createSubscriptionSQL = `select create_subscription($1::integer, $2::varchar)`

func InsertSubscription(ctx context.Context, db pgxutil.DB, userID int32, feedURL string) error {
	_, err := db.Exec(ctx, createSubscriptionSQL, userID, feedURL)
	return err
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
group by feeds.id
order by name`

func SelectSubscriptions(ctx context.Context, db pgxutil.DB, userID int32) ([]Subscription, error) {
	subs := make([]Subscription, 0, 16)
	rows, _ := db.Query(ctx, getSubscriptionsSQL, userID)
	for rows.Next() {
		var s Subscription
		rows.Scan(&s.FeedID, &s.Name, &s.URL, &s.LastFetchTime, &s.LastFailure, &s.LastFailureTime, &s.FailureCount, &s.ItemCount, &s.LastPublicationTime)
		subs = append(subs, s)
	}

	return subs, rows.Err()
}

const deleteSubscriptionSQL = `delete from subscriptions where user_id=$1 and feed_id=$2`
const deleteFeedIfOrphanedSQL = `delete from feeds
where id=$1
  and not exists(select 1 from subscriptions where feed_id=id)`

func DeleteSubscription(ctx context.Context, db *pgxpool.Pool, userID, feedID int32) error {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, deleteSubscriptionSQL, userID, feedID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, deleteFeedIfOrphanedSQL, feedID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
