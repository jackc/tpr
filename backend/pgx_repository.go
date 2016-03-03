package main

import (
	"fmt"

	"github.com/jackc/pgx"
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
