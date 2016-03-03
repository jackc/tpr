package data

import (
	"bytes"
	"io"
	"strconv"
	"time"

	"github.com/jackc/pgx"
)

func MarkItemRead(db Queryer, userID, itemID int32) error {
	commandTag, err := db.Exec("markItemRead", userID, itemID)
	if err != nil {
		return err
	}
	if commandTag != "DELETE 1" {
		return ErrNotFound
	}

	return nil
}

func CopySubscriptionsForUserAsJSON(db Queryer, w io.Writer, userID int32) error {
	var b []byte
	err := db.QueryRow("getFeedsForUser", userID).Scan(&b)
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

func CopyUnreadItemsAsJSONByUserID(db Queryer, w io.Writer, userID int32) error {
	var b []byte
	err := db.QueryRow("getUnreadItems", userID).Scan(&b)
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

func UpdateFeedWithFetchSuccess(db *pgx.ConnPool, feedID int32, update *ParsedFeed, etag String, fetchTime time.Time) error {
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

func UpdateFeedWithFetchUnchanged(db Queryer, feedID int32, fetchTime time.Time) (err error) {
	_, err = db.Exec("updateFeedWithFetchUnchanged", fetchTime, feedID)
	return
}

func UpdateFeedWithFetchFailure(db Queryer, feedID int32, failure string, fetchTime time.Time) (err error) {
	_, err = db.Exec("updateFeedWithFetchFailure", failure, fetchTime, feedID)
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
