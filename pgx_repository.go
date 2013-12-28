package main

import (
	"bytes"
	"fmt"
	"github.com/JackC/pgx"
	"io"
	"strconv"
	"time"
)

type pgxRepository struct {
	pool *pgx.ConnectionPool
}

func NewPgxRepository(parameters pgx.ConnectionParameters, options pgx.ConnectionPoolOptions) (*pgxRepository, error) {
	pool, err := pgx.NewConnectionPool(parameters, options)
	if err != nil {
		return nil, err
	}

	repo := &pgxRepository{pool: pool}
	return repo, nil
}

func (repo *pgxRepository) createUser(name string, passwordDigest, passwordSalt []byte) (int32, error) {
	v, err := repo.pool.SelectValue("insertUser", name, passwordDigest, passwordSalt)
	if err != nil {
		return 0, err
	}
	userID := v.(int32)

	return userID, err
}

func (repo *pgxRepository) getUserAuthenticationByName(name string) (userID int32, passwordDigest, passwordSalt []byte, err error) {
	err = repo.pool.SelectFunc("getUserAuthenticationByName", func(r *pgx.DataRowReader) (err error) {
		userID = r.ReadValue().(int32)
		passwordDigest = r.ReadValue().([]byte)
		passwordSalt = r.ReadValue().([]byte)
		return
	}, name)

	return
}

func (repo *pgxRepository) getUserName(userID int32) (name string, err error) {
	v, err := repo.pool.SelectValue("getUserName", userID)
	name = v.(string)
	return
}

func (repo *pgxRepository) createFeed(name, url string) (int32, error) {
	feedID, err := repo.pool.SelectValue("insertFeed", name, url)
	if err != nil {
		return 0, err
	}

	return feedID.(int32), err
}

func (repo *pgxRepository) getFeedIDByURL(url string) (feedID int32, err error) {
	var id interface{}
	id, err = repo.pool.SelectValue("getFeedIDByURL", url)
	if _, ok := err.(pgx.NotSingleRowError); ok {
		return 0, notFound
	}
	if err != nil {
		return 0, err
	}

	return id.(int32), nil
}

func (repo *pgxRepository) getFeedsUncheckedSince(since time.Time) (feeds []staleFeed, err error) {
	err = repo.pool.SelectFunc("getFeedsUncheckedSince", func(r *pgx.DataRowReader) (err error) {
		var feed staleFeed
		feed.id = r.ReadValue().(int32)
		feed.url = r.ReadValue().(string)
		etag := r.ReadValue()
		feed.etag, _ = etag.(string) // ignore if null
		feeds = append(feeds, feed)
		return
	}, since)

	return
}

func (repo *pgxRepository) updateFeedWithFetchSuccess(feedID int32, update *parsedFeed, etag string, fetchTime time.Time) (err error) {
	var conn *pgx.Connection

	conn, err = repo.pool.Acquire()
	if err != nil {
		return
	}
	defer repo.pool.Release(conn)

	conn.Transaction(func() bool {
		_, err = conn.Execute("updateFeedWithFetchSuccess",
			update.name,
			fetchTime,
			etag,
			feedID)
		if err != nil {
			return false
		}

		if len(update.items) > 0 {
			insertSQL, insertArgs := repo.buildNewItemsSQL(feedID, update.items)
			_, err = conn.Execute(insertSQL, insertArgs...)
			if err != nil {
				return false
			}
		}

		return true
	})

	return
}

func (repo *pgxRepository) updateFeedWithFetchUnchanged(feedID int32, fetchTime time.Time) (err error) {
	_, err = repo.pool.Execute("updateFeedWithFetchUnchanged", fetchTime, feedID)
	return
}

func (repo *pgxRepository) updateFeedWithFetchFailure(feedID int32, failure string, fetchTime time.Time) (err error) {
	_, err = repo.pool.Execute("updateFeedWithFetchFailure", failure, fetchTime, feedID)
	return err
}

func (repo *pgxRepository) copyFeedsAsJSONBySubscribedUserID(w io.Writer, userID int32) error {
	return repo.pool.SelectValueTo(w, "getFeedsForUser", userID)
}

func (repo *pgxRepository) copyUnreadItemsAsJSONByUserID(w io.Writer, userID int32) error {
	return repo.pool.SelectValueTo(w, "getUnreadItems", userID)
}

func (repo *pgxRepository) markItemRead(userID, itemID int32) error {
	commandTag, err := repo.pool.Execute("markItemRead", userID, itemID)
	if err != nil {
		return err
	}
	if commandTag != "DELETE 1" {
		return notFound
	}

	return nil
}

// TODO - change interface to only mark items read that are visible to user when they issue command
func (repo *pgxRepository) markAllItemsRead(userID int32) error {
	_, err := repo.pool.Execute("markAllItemsRead", userID)
	return err
}

func (repo *pgxRepository) createSubscription(userID, feedID int32) error {
	_, err := repo.pool.Execute("insertSubscription", userID, feedID)
	return err
}

func (repo *pgxRepository) createSession(id []byte, userID int32) (err error) {
	_, err = repo.pool.Execute("insertSession", id, userID)
	return err
}

func (repo *pgxRepository) getUserIDBySessionID(id []byte) (userID int32, err error) {
	v, err := repo.pool.SelectValue("getUserIDBySessionID", id)
	if _, ok := err.(pgx.NotSingleRowError); ok {
		return 0, notFound
	}
	if err != nil {
		return 0, err
	}
	return v.(int32), err
}

func (repo *pgxRepository) deleteSession(id []byte) error {
	commandTag, err := repo.pool.Execute("deleteSession", id)
	if err != nil {
		return err
	}
	if commandTag != "DELETE 1" {
		return notFound
	}

	return nil
}

// Empty all data in the entire repository
func (repo *pgxRepository) empty() error {
	tables := []string{"feeds", "items", "sessions", "subscriptions", "unread_items", "users"}
	for _, table := range tables {
		_, err := repo.pool.Execute(fmt.Sprintf("truncate %s restart identity cascade", table))
		if err != nil {
			return err
		}
	}
	return nil
}

func (repo *pgxRepository) buildNewItemsSQL(feedID int32, items []parsedItem) (sql string, args []interface{}) {
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

// afterConnect creates the prepared statements that this application uses
func afterConnect(conn *pgx.Connection) (err error) {
	err = conn.Prepare("getUnreadItems", `
    select coalesce(json_agg(row_to_json(t)), '[]'::json)
    from (
      select
        items.id,
        feeds.id as feed_id,
        feeds.name as feed_name,
        items.title,
        items.url,
        publication_time
      from feeds
        join items on feeds.id=items.feed_id
        join unread_items on items.id=unread_items.item_id
      where user_id=$1
      order by publication_time asc
    ) t`)
	if err != nil {
		return
	}

	err = conn.Prepare("deleteSession", `delete from sessions where id=$1`)
	if err != nil {
		return
	}

	err = conn.Prepare("getFeedsForUser", `
    select json_agg(row_to_json(t))
    from (
      select feeds.name, feeds.url, feeds.last_fetch_time
      from feeds
        join subscriptions on feeds.id=subscriptions.feed_id
      where user_id=$1
      order by name
    ) t`)
	if err != nil {
		return
	}

	err = conn.Prepare("markItemRead", `
    delete from unread_items
    where user_id=$1
      and item_id=$2`)
	if err != nil {
		return
	}

	err = conn.Prepare("markAllItemsRead", `delete from unread_items where user_id=$1`)
	if err != nil {
		return
	}

	err = conn.Prepare("insertUser", `
    insert into users(name, password_digest, password_salt)
    values($1, $2, $3)
    returning id`)
	if err != nil {
		return
	}

	err = conn.Prepare("getUserAuthenticationByName", `
    select id, password_digest, password_salt from users where name=$1`)
	if err != nil {
		return
	}

	err = conn.Prepare("insertFeed", `
    insert into feeds(name, url) values($1, $2) returning id`)
	if err != nil {
		return
	}

	err = conn.Prepare("getFeedIDByURL", `select id from feeds where url=$1`)
	if err != nil {
		return
	}

	err = conn.Prepare("getFeedsUncheckedSince", `
    select id, url, etag
    from feeds
    where greatest(last_fetch_time, last_failure_time, '-Infinity'::timestamptz) < $1`)
	if err != nil {
		return
	}

	err = conn.Prepare("updateFeedWithFetchSuccess", `
      update feeds
      set name=$1,
        last_fetch_time=$2,
        etag=$3,
        last_failure=null,
        last_failure_time=null,
        failure_count=0
      where id=$4`)
	if err != nil {
		return
	}

	err = conn.Prepare("updateFeedWithFetchUnchanged", `
    update feeds
    set last_fetch_time=$1,
      last_failure=null,
      last_failure_time=null,
      failure_count=0
    where id=$2`)
	if err != nil {
		return
	}

	err = conn.Prepare("updateFeedWithFetchFailure", `
    update feeds
    set last_failure=$1,
      last_failure_time=$2,
      failure_count=failure_count+1
    where id=$3`)
	if err != nil {
		return
	}

	err = conn.Prepare("insertSubscription", `insert into subscriptions(user_id, feed_id) values($1, $2)`)
	if err != nil {
		return
	}

	err = conn.Prepare("insertSession", `insert into sessions(id, user_id) values($1, $2)`)
	if err != nil {
		return
	}

	err = conn.Prepare("getUserIDBySessionID", `select user_id from sessions where id=$1`)
	if err != nil {
		return
	}

	err = conn.Prepare("getUserName", `select name from users where id=$1`)
	if err != nil {
		return
	}

	return
}
