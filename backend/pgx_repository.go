package main

import (
	"bytes"
	"fmt"
	"github.com/jackc/pgx"
	"github.com/jackc/tpr/backend/data"
	"io"
	"strconv"
	"strings"
	"time"
)

type pgxRepository struct {
	pool *pgx.ConnPool
}

func NewPgxRepository(connPoolConfig pgx.ConnPoolConfig) (*pgxRepository, error) {
	pool, err := pgx.NewConnPool(connPoolConfig)
	if err != nil {
		return nil, err
	}

	repo := &pgxRepository{pool: pool}
	return repo, nil
}

func (repo *pgxRepository) CreateUser(user *data.User) (int32, error) {
	err := data.InsertUser(repo.pool, user)
	if err != nil {
		if strings.Contains(err.Error(), "users_name_unq") {
			return 0, DuplicationError{Field: "name"}
		}
		if strings.Contains(err.Error(), "users_email_key") {
			return 0, DuplicationError{Field: "email"}
		}
		return 0, err
	}

	return user.ID.Value, nil
}

func (repo *pgxRepository) getUser(sql string, arg interface{}) (*data.User, error) {
	user := data.User{}

	err := repo.pool.QueryRow(sql, arg).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordDigest, &user.PasswordSalt)
	if err == pgx.ErrNoRows {
		return nil, notFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (repo *pgxRepository) GetUser(userID int32) (*data.User, error) {
	return repo.getUser("getUser", userID)
}

func (repo *pgxRepository) GetUserByName(name string) (*data.User, error) {
	return repo.getUser("getUserByName", name)
}

func (repo *pgxRepository) GetUserByEmail(email string) (*data.User, error) {
	return repo.getUser("getUserByEmail", email)
}

func (repo *pgxRepository) UpdateUser(userID int32, attributes *data.User) error {
	err := data.UpdateUser(repo.pool, userID, attributes)
	if err == data.ErrNotFound {
		return notFound
	}
	return err
}

func (repo *pgxRepository) GetFeedsUncheckedSince(since time.Time) ([]data.Feed, error) {
	feeds := make([]data.Feed, 0, 8)
	rows, _ := repo.pool.Query("getFeedsUncheckedSince", since)

	for rows.Next() {
		var feed data.Feed
		rows.Scan(&feed.ID, &feed.URL, &feed.ETag)
		feeds = append(feeds, feed)
	}

	return feeds, rows.Err()
}

func (repo *pgxRepository) UpdateFeedWithFetchSuccess(feedID int32, update *parsedFeed, etag data.String, fetchTime time.Time) error {
	tx, err := repo.pool.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("updateFeedWithFetchSuccess",
		update.name,
		fetchTime,
		&etag,
		feedID)
	if err != nil {
		return err
	}

	if len(update.items) > 0 {
		insertSQL, insertArgs := repo.buildNewItemsSQL(feedID, update.items)
		_, err = tx.Exec(insertSQL, insertArgs...)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (repo *pgxRepository) UpdateFeedWithFetchUnchanged(feedID int32, fetchTime time.Time) (err error) {
	_, err = repo.pool.Exec("updateFeedWithFetchUnchanged", fetchTime, feedID)
	return
}

func (repo *pgxRepository) UpdateFeedWithFetchFailure(feedID int32, failure string, fetchTime time.Time) (err error) {
	_, err = repo.pool.Exec("updateFeedWithFetchFailure", failure, fetchTime, feedID)
	return err
}

func (repo *pgxRepository) CopySubscriptionsForUserAsJSON(w io.Writer, userID int32) error {
	var b []byte
	err := repo.pool.QueryRow("getFeedsForUser", userID).Scan(&b)
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

func (repo *pgxRepository) CopyUnreadItemsAsJSONByUserID(w io.Writer, userID int32) error {
	var b []byte
	err := repo.pool.QueryRow("getUnreadItems", userID).Scan(&b)
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

func (repo *pgxRepository) MarkItemRead(userID, itemID int32) error {
	commandTag, err := repo.pool.Exec("markItemRead", userID, itemID)
	if err != nil {
		return err
	}
	if commandTag != "DELETE 1" {
		return notFound
	}

	return nil
}

func (repo *pgxRepository) CreateSubscription(userID int32, feedURL string) error {
	_, err := repo.pool.Exec("createSubscription", userID, feedURL)
	return err
}

func (repo *pgxRepository) GetSubscriptions(userID int32) ([]Subscription, error) {
	subs := make([]Subscription, 0, 16)
	rows, _ := repo.pool.Query("getSubscriptions", userID)
	for rows.Next() {
		var s Subscription
		rows.Scan(&s.FeedID, &s.Name, &s.URL, &s.LastFetchTime, &s.LastFailure, &s.LastFailureTime, &s.FailureCount, &s.ItemCount, &s.LastPublicationTime)
		subs = append(subs, s)
	}

	return subs, rows.Err()
}

func (repo *pgxRepository) DeleteSubscription(userID, feedID int32) error {
	tx, err := repo.pool.BeginIso(pgx.Serializable)
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

func (repo *pgxRepository) CreateSession(id []byte, userID int32) (err error) {
	_, err = repo.pool.Exec("insertSession", id, userID)
	return err
}

func (repo *pgxRepository) GetUserBySessionID(id []byte) (*data.User, error) {
	return repo.getUser("getUserBySessionID", id)
}

func (repo *pgxRepository) DeleteSession(id []byte) error {
	commandTag, err := repo.pool.Exec("deleteSession", id)
	if err != nil {
		return err
	}
	if commandTag != "DELETE 1" {
		return notFound
	}

	return nil
}

func (repo *pgxRepository) CreatePasswordReset(attrs *data.PasswordReset) error {
	err := data.InsertPasswordReset(repo.pool, attrs)
	if err != nil {
		if strings.Contains(err.Error(), "password_resets_pkey") {
			return DuplicationError{Field: "token"}
		}
		return err
	}

	return nil
}

func (repo *pgxRepository) GetPasswordReset(token string) (*data.PasswordReset, error) {
	pwr, err := data.SelectPasswordResetByPK(repo.pool, token)
	if err == data.ErrNotFound {
		return nil, notFound
	}
	return pwr, err
}

func (repo *pgxRepository) UpdatePasswordReset(token string, attrs *data.PasswordReset) error {
	err := data.UpdatePasswordReset(repo.pool, token, attrs)
	if err == data.ErrNotFound {
		return notFound
	}
	return err
}

// Empty all data in the entire repository
func (repo *pgxRepository) empty() error {
	tables := []string{"feeds", "items", "password_resets", "sessions", "subscriptions", "unread_items", "users"}
	for _, table := range tables {
		_, err := repo.pool.Exec(fmt.Sprintf("delete from %s", table))
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
		if item.publicationTime.Status == data.Present {
			args = append(args, item.publicationTime.Value)
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

// afterConnect creates the prepared statements that this application uses
func afterConnect(conn *pgx.Conn) (err error) {
	_, err = conn.Prepare("getUnreadItems", `
    select coalesce(json_agg(row_to_json(t)), '[]'::json)
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
    ) t`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("deleteSession", `delete from sessions where id=$1`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("getFeedsForUser", `
    select coalesce(json_agg(row_to_json(t)), '[]'::json)
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
    ) t`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("markItemRead", `
    delete from unread_items
    where user_id=$1
      and item_id=$2`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("insertUser", `
    insert into users(name, email, password_digest, password_salt)
    values($1, $2, $3, $4)
    returning id`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("getUserAuthenticationByName", `
    select id, password_digest, password_salt from users where name=$1`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("getFeedsUncheckedSince", `
    select id, url, etag
    from feeds
    where greatest(last_fetch_time, last_failure_time, '-Infinity'::timestamptz) < $1`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("updateFeedWithFetchSuccess", `
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

	_, err = conn.Prepare("updateFeedWithFetchUnchanged", `
    update feeds
    set last_fetch_time=$1,
      last_failure=null,
      last_failure_time=null,
      failure_count=0
    where id=$2`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("updateFeedWithFetchFailure", `
    update feeds
    set last_failure=$1,
      last_failure_time=$2,
      failure_count=failure_count+1
    where id=$3`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("createSubscription", `select create_subscription($1::integer, $2::varchar)`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("getSubscriptions", `
    select feeds.id as feed_id,
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
    order by name`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("deleteSubscription", `delete from subscriptions where user_id=$1 and feed_id=$2`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("deleteFeedIfOrphaned", `
    delete from feeds
    where id=$1
      and not exists(select 1 from subscriptions where feed_id=id)`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("insertSession", `insert into sessions(id, user_id) values($1, $2)`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("getUserBySessionID", `
    select users.id, name, email, password_digest, password_salt
    from sessions
      join users on sessions.user_id=users.id
    where sessions.id=$1`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("getUserName", `select name from users where id=$1`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("getUser", `select id, name, email, password_digest, password_salt from users where id=$1`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("getUserByName", `select id, name, email, password_digest, password_salt from users where name=$1`)
	if err != nil {
		return
	}

	_, err = conn.Prepare("getUserByEmail", `select id, name, email, password_digest, password_salt from users where email=$1`)
	if err != nil {
		return
	}

	return
}
