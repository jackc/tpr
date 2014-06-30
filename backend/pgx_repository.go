package main

import (
	"bytes"
	"fmt"
	"github.com/jackc/box"
	"github.com/jackc/pgx"
	"io"
	"strconv"
	"strings"
	"time"
)

type pgxRepository struct {
	pool *pgx.ConnPool
}

func NewPgxRepository(connPoolConfig pgx.ConnPoolConfig) (*pgxRepository, error) {
	connPoolConfig.MsgBufSize = 16 * 1024
	pool, err := pgx.NewConnPool(connPoolConfig)
	if err != nil {
		return nil, err
	}

	repo := &pgxRepository{pool: pool}
	return repo, nil
}

func (repo *pgxRepository) CreateUser(user *User) (int32, error) {
	v, err := repo.pool.SelectValue("insertUser",
		user.Name.GetCoerceNil(),
		user.Email.GetCoerceNil(),
		user.PasswordDigest,
		user.PasswordSalt)
	if err != nil {
		if strings.Contains(err.Error(), "users_name_unq") {
			return 0, DuplicationError{Field: "name"}
		}
		if strings.Contains(err.Error(), "users_email_key") {
			return 0, DuplicationError{Field: "email"}
		}
		return 0, err
	}

	return v.(int32), nil
}

func (repo *pgxRepository) getUser(sql string, arg interface{}) (*User, error) {
	user := User{}

	qr, _ := repo.pool.Query(sql, arg)
	for qr.NextRow() {
		var rr pgx.RowReader
		user.ID.Set(rr.ReadInt32(qr))
		user.Name.Set(rr.ReadString(qr))
		user.Email.SetCoerceNil(rr.ReadValue(qr), box.Empty)
		user.PasswordDigest = rr.ReadValue(qr).([]byte)
		user.PasswordSalt = rr.ReadValue(qr).([]byte)
	}

	if qr.Err() != nil {
		return nil, qr.Err()
	}

	if _, ok := user.ID.Get(); !ok {
		return nil, notFound
	}

	return &user, nil
}

func (repo *pgxRepository) GetUser(userID int32) (*User, error) {
	return repo.getUser("getUser", userID)
}

func (repo *pgxRepository) GetUserByName(name string) (*User, error) {
	return repo.getUser("getUserByName", name)
}

func (repo *pgxRepository) GetUserByEmail(email string) (*User, error) {
	return repo.getUser("getUserByEmail", email)
}

func (repo *pgxRepository) UpdateUser(userID int32, attributes *User) error {
	var sets []string
	args := pgx.QueryArgs(make([]interface{}, 0, 6))

	if v, ok := attributes.ID.Get(); ok {
		sets = append(sets, "id="+args.Append(v))
	}
	if v, ok := attributes.Name.Get(); ok {
		sets = append(sets, "name="+args.Append(v))
	}
	if v, ok := attributes.Email.Get(); ok {
		sets = append(sets, "email="+args.Append(v))
	}
	if attributes.PasswordDigest != nil {
		sets = append(sets, "password_digest="+args.Append(attributes.PasswordDigest))
	}
	if attributes.PasswordSalt != nil {
		sets = append(sets, "password_salt="+args.Append(attributes.PasswordSalt))
	}

	sql := "update users set " + strings.Join(sets, ", ") + " where id=" + args.Append(userID)

	commandTag, err := repo.pool.Exec(sql, args...)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return notFound
	}
	return nil
}

func (repo *pgxRepository) GetFeedsUncheckedSince(since time.Time) ([]Feed, error) {
	feeds := make([]Feed, 0, 8)
	qr, _ := repo.pool.Query("getFeedsUncheckedSince", since)

	for qr.NextRow() {
		var rr pgx.RowReader
		var feed Feed
		feed.ID.Set(rr.ReadInt32(qr))
		feed.URL.Set(rr.ReadString(qr))
		feed.ETag.SetCoerceNil(rr.ReadValue(qr), box.Empty)
		feeds = append(feeds, feed)
	}

	return feeds, qr.Err()
}

func (repo *pgxRepository) UpdateFeedWithFetchSuccess(feedID int32, update *parsedFeed, etag box.String, fetchTime time.Time) (err error) {
	var conn *pgx.Conn

	conn, err = repo.pool.Acquire()
	if err != nil {
		return
	}
	defer repo.pool.Release(conn)

	conn.Transaction(func() bool {
		_, err = conn.Exec("updateFeedWithFetchSuccess",
			update.name,
			fetchTime,
			etag.GetCoerceNil(),
			feedID)
		if err != nil {
			return false
		}

		if len(update.items) > 0 {
			insertSQL, insertArgs := repo.buildNewItemsSQL(feedID, update.items)
			_, err = conn.Exec(insertSQL, insertArgs...)
			if err != nil {
				return false
			}
		}

		return true
	})

	return
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
	return repo.pool.SelectValueTo(w, "getFeedsForUser", userID)
}

func (repo *pgxRepository) CopyUnreadItemsAsJSONByUserID(w io.Writer, userID int32) error {
	return repo.pool.SelectValueTo(w, "getUnreadItems", userID)
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
	qr, _ := repo.pool.Query("getSubscriptions", userID)
	for qr.NextRow() {
		var rr pgx.RowReader
		var s Subscription
		s.FeedID.Set(rr.ReadInt32(qr))
		s.Name.Set(rr.ReadString(qr))
		s.URL.Set(rr.ReadString(qr))
		s.LastFetchTime.SetCoerceNil(rr.ReadValue(qr), box.Empty)
		s.LastFailure.SetCoerceNil(rr.ReadValue(qr), box.Empty)
		s.LastFailureTime.SetCoerceNil(rr.ReadValue(qr), box.Empty)
		s.FailureCount.Set(rr.ReadInt32(qr))
		s.ItemCount.Set(rr.ReadInt64(qr))
		s.LastPublicationTime.SetCoerceNil(rr.ReadValue(qr), box.Empty)
		subs = append(subs, s)
	}

	return subs, qr.Err()
}

func (repo *pgxRepository) DeleteSubscription(userID, feedID int32) error {
	conn, err := repo.pool.Acquire()
	if err != nil {
		return err
	}
	defer repo.pool.Release(conn)

	conn.TransactionIso("serializable", func() bool {
		_, err := conn.Exec("deleteSubscription", userID, feedID)
		if err != nil {
			return false
		}

		_, err = conn.Exec("deleteFeedIfOrphaned", feedID)
		if err != nil {
			return false
		}

		return true
	})

	return err
}

func (repo *pgxRepository) CreateSession(id []byte, userID int32) (err error) {
	_, err = repo.pool.Exec("insertSession", id, userID)
	return err
}

func (repo *pgxRepository) GetUserBySessionID(id []byte) (*User, error) {
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

func (repo *pgxRepository) CreatePasswordReset(attrs *PasswordReset) error {
	columns := make([]string, 0, 7)
	placeholders := make([]string, 0, 7)
	args := pgx.QueryArgs(make([]interface{}, 0, 7))

	if v, ok := attrs.Token.Get(); ok {
		columns = append(columns, "token")
		placeholders = append(placeholders, args.Append(v))
	}
	if v, ok := attrs.Email.Get(); ok {
		columns = append(columns, "email")
		placeholders = append(placeholders, args.Append(v))
	}
	if v, ok := attrs.RequestIP.Get(); ok {
		columns = append(columns, "request_ip")
		placeholders = append(placeholders, args.Append(v))
	}
	if v, ok := attrs.RequestTime.Get(); ok {
		columns = append(columns, "request_time")
		placeholders = append(placeholders, args.Append(v))
	}
	if v, ok := attrs.UserID.Get(); ok {
		columns = append(columns, "user_id")
		placeholders = append(placeholders, args.Append(v))
	}
	if v, ok := attrs.CompletionTime.Get(); ok {
		columns = append(columns, "completion_time")
		placeholders = append(placeholders, args.Append(v))
	}
	if v, ok := attrs.CompletionIP.Get(); ok {
		columns = append(columns, "completion_ip")
		placeholders = append(placeholders, args.Append(v))
	}

	sql := "insert into password_resets(" + strings.Join(columns, ", ") + ") values(" + strings.Join(placeholders, ", ") + ")"

	_, err := repo.pool.Exec(sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "password_resets_pkey") {
			return DuplicationError{Field: "token"}
		}
		return err
	}

	return nil
}

func (repo *pgxRepository) GetPasswordReset(token string) (*PasswordReset, error) {
	pwr := &PasswordReset{}
	qr, _ := repo.pool.Query("getPasswordReset", token)
	for qr.NextRow() {
		var rr pgx.RowReader
		pwr.Token.Set(rr.ReadString(qr))
		pwr.Email.Set(rr.ReadString(qr))
		pwr.RequestIP.SetCoerceNil(rr.ReadValue(qr), box.Unknown)
		pwr.RequestTime.Set(rr.ReadTime(qr))
		pwr.UserID.SetCoerceNil(rr.ReadValue(qr), box.Empty)
		pwr.CompletionIP.SetCoerceNil(rr.ReadValue(qr), box.Empty)
		pwr.CompletionTime.SetCoerceNil(rr.ReadValue(qr), box.Empty)
	}
	if qr.Err() != nil {
		return nil, qr.Err()
	}

	if _, ok := pwr.Token.Get(); !ok {
		return nil, notFound
	}

	return pwr, nil
}

func (repo *pgxRepository) UpdatePasswordReset(token string, attrs *PasswordReset) error {
	sets := make([]string, 0, 7)
	args := pgx.QueryArgs(make([]interface{}, 0, 7))

	if v, ok := attrs.Token.Get(); ok {
		sets = append(sets, "token="+args.Append(v))
	}
	if v, ok := attrs.Email.Get(); ok {
		sets = append(sets, "email="+args.Append(v))
	}
	if v, ok := attrs.RequestIP.Get(); ok {
		sets = append(sets, "request_ip="+args.Append(v))
	}
	if v, ok := attrs.RequestTime.Get(); ok {
		sets = append(sets, "request_time="+args.Append(v))
	}
	if v, ok := attrs.CompletionTime.Get(); ok {
		sets = append(sets, "completion_time="+args.Append(v))
	}
	if v, ok := attrs.CompletionIP.Get(); ok {
		sets = append(sets, "completion_ip="+args.Append(v))
	}

	sql := "update password_resets set " + strings.Join(sets, ", ") + " where token=" + args.Append(token)

	commandTag, err := repo.pool.Exec(sql, args...)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return notFound
	}
	return nil
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
		args = append(args, item.publicationTime.GetCoerceNil())
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

	_, err = conn.Prepare("getPasswordReset", `
    select token, email, request_ip, request_time, user_id, completion_ip, completion_time
    from password_resets
    where token=$1`)
	if err != nil {
		return
	}

	return
}
