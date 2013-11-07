package main

import (
	"fmt"
	"github.com/JackC/pgx"
	mig "github.com/JackC/pgx/migrate"
)

func migrate(connectionParameters pgx.ConnectionParameters) (err error) {
	var conn *pgx.Connection
	conn, err = pgx.Connect(connectionParameters)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := conn.Close()
		if err == nil {
			err = closeErr
		}
	}()

	var m *mig.Migrator
	m, err = mig.NewMigrator(conn, "schema_version")
	if err != nil {
		return
	}

	m.OnStart = func(migration *mig.Migration) {
		fmt.Printf("Migrating %d: %s\n", migration.Sequence, migration.Name)
	}

	m.AppendMigration("Create users", `
    create table users(
      id serial primary key,
      name varchar(30) not null check(name ~ '\A[a-zA-Z0-9]+\Z'),
      password_digest bytea not null,
      password_salt bytea not null
    );

    create unique index users_name_unq on users (lower(name));
  `)

	m.AppendMigration("Create feeds", `
    create table feeds(
      id serial primary key,
      name varchar not null check(name<>''),
      url varchar not null unique check(url<>''),
      last_fetch_time timestamp with time zone,
      etag varchar,
      last_failure varchar,
      last_failure_time timestamp with time zone,
      failure_count integer not null default 0,
      creation_time timestamp with time zone not null default now()
    );

    create index on feeds (last_fetch_time);
  `)

	m.AppendMigration("Create items", `
    create extension pgcrypto;

    create table items(
      id serial primary key,
      feed_id integer not null references feeds,
      publication_time timestamp with time zone,
      title varchar not null,
      url varchar not null,
      digest bytea not null unique,
      creation_time timestamp with time zone not null default now(),
      unique(feed_id, id)
    );

    create index on items (feed_id);

    create function digest_item(feed_id integer, publication_time timestamp with time zone, title text, url text) returns bytea as $$
      begin
        return digest(feed_id::text || publication_time::text || title || url, 'sha256');
        return new;
      end;
    $$ language plpgsql;

    create function digest_items() returns trigger as $$
      begin
        new.digest := digest_item(new.feed_id, new.publication_time, new.title, new.url);
        return new;
      end;
    $$ language plpgsql;

    create trigger digest_items before insert or update on items
      for each row execute procedure digest_items();
  `)

	m.AppendMigration("Create subscriptions", `
    create table subscriptions(
      user_id integer not null,
      feed_id integer not null,
      primary key(user_id, feed_id)
    );

    create index on subscriptions (feed_id);
  `)

	m.AppendMigration("Create unread_items", `
    create table unread_items(
      user_id integer not null,
      feed_id integer not null,
      item_id integer not null,
      primary key(user_id, feed_id, item_id),
      foreign key (user_id, feed_id) references subscriptions (user_id, feed_id) on delete cascade,
      foreign key (feed_id, item_id) references items (feed_id, id) on delete cascade
    );
  `)

	return m.Migrate()
}
