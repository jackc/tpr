package main

import (
	"fmt"
	"github.com/JackC/pgx"
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
	v, err := repo.pool.SelectValue("insert into users(name, password_digest, password_salt) values($1, $2, $3) returning id", name, passwordDigest, passwordSalt)
	if err != nil {
		return 0, err
	}
	userID := v.(int32)

	return userID, err
}

func (repo *pgxRepository) getUserAuthenticationByName(name string) (userID int32, passwordDigest, passwordSalt []byte, err error) {
	err = repo.pool.SelectFunc("select id, password_digest, password_salt from users where name=$1", func(r *pgx.DataRowReader) (err error) {
		userID = r.ReadValue().(int32)
		passwordDigest = r.ReadValue().([]byte)
		passwordSalt = r.ReadValue().([]byte)
		return
	}, name)

	return
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
      and item_id=$2
    returning item_id`)
	if err != nil {
		return
	}

	err = conn.Prepare("markAllItemsRead", `delete from unread_items where user_id=$1`)
	if err != nil {
		return
	}

	return
}
