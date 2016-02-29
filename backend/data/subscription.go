package data

import (
	"github.com/jackc/pgx"
)

type Subscription struct {
	FeedID              Int32
	Name                String
	URL                 String
	LastFetchTime       Time
	LastFailure         String
	LastFailureTime     Time
	FailureCount        Int32
	ItemCount           Int64
	LastPublicationTime Time
}

func InsertSubscription(db Queryer, userID int32, feedURL string) error {
	_, err := db.Exec("createSubscription", userID, feedURL)
	return err
}

func SelectSubscriptions(db Queryer, userID int32) ([]Subscription, error) {
	subs := make([]Subscription, 0, 16)
	rows, _ := db.Query("getSubscriptions", userID)
	for rows.Next() {
		var s Subscription
		rows.Scan(&s.FeedID, &s.Name, &s.URL, &s.LastFetchTime, &s.LastFailure, &s.LastFailureTime, &s.FailureCount, &s.ItemCount, &s.LastPublicationTime)
		subs = append(subs, s)
	}

	return subs, rows.Err()
}

func DeleteSubscription(db *pgx.ConnPool, userID, feedID int32) error {
	tx, err := db.BeginIso(pgx.Serializable)
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
