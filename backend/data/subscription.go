package data

import (
	"context"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
)

type Subscription struct {
	FeedID              pgtype.Int4
	Name                pgtype.Varchar
	URL                 pgtype.Varchar
	LastFetchTime       pgtype.Timestamptz
	LastFailure         pgtype.Varchar
	LastFailureTime     pgtype.Timestamptz
	FailureCount        pgtype.Int4
	ItemCount           pgtype.Int8
	LastPublicationTime pgtype.Timestamptz
}

const createSubscriptionSQL = `select create_subscription($1::integer, $2::varchar)`

func InsertSubscription(db Queryer, userID int32, feedURL string) error {
	_, err := prepareExec(db, "createSubscription", createSubscriptionSQL, userID, feedURL)
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

func SelectSubscriptions(db Queryer, userID int32) ([]Subscription, error) {
	subs := make([]Subscription, 0, 16)
	rows, _ := prepareQuery(db, "getSubscriptions", getSubscriptionsSQL, userID)
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

func DeleteSubscription(db *pgx.ConnPool, userID, feedID int32) error {
	if _, err := db.Prepare("deleteSubscription", deleteSubscriptionSQL); err != nil {
		return err
	}
	if _, err := db.Prepare("deleteFeedIfOrphaned", deleteFeedIfOrphanedSQL); err != nil {
		return err
	}

	tx, err := db.BeginEx(context.Background(), &pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("deleteSubscription", userID, feedID)
	if err != nil {
		return err
	}

	_, err = tx.Exec("deleteFeedIfOrphaned", feedID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
