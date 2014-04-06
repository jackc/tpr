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
    create extension if not exists pgcrypto;

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

	m.AppendMigration("Create sessions", `
    create unlogged table sessions(
      id bytea primary key,
      user_id integer not null references users on delete cascade,
      start_time timestamp with time zone not null default now()
    );
  `)

	m.AppendMigration("Create create_subscription", `
    create function create_subscription(user_id integer, url varchar) returns void as
    $$
    declare
      feed_id integer;
    begin
      loop
        -- try to find existing feed
        select id into feed_id from feeds where feeds.url=create_subscription.url;
        if found then
          exit;
        end if;

        -- if feed was not found then create it
        begin
          insert into feeds(name, url)
            values(create_subscription.url, create_subscription.url)
            returning id into feed_id;

          exit;
        exception when unique_violation then
          -- try again
        end;
      end loop;

      insert into subscriptions(user_id, feed_id) values(user_id, feed_id);
    end;
    $$
    language plpgsql;
  `)

	m.AppendMigration("Create subscriptions foreign key constraints", `
    alter table subscriptions
      add constraint subscriptions_user_id_fkey foreign key (user_id) references users on delete cascade,
      add constraint subscriptions_feed_id_fkey foreign key (feed_id) references feeds on delete cascade;
  `)

	m.AppendMigration("Alter items feed_id FK constraint to cascade delete", `
    alter table items
      drop constraint items_feed_id_fkey,
      add constraint items_feed_id_fkey foreign key (feed_id) references feeds on delete cascade;
  `)

	m.AppendMigration("Alter digest_item to allow for null publication_time", `
    create or replace function digest_item(feed_id integer, publication_time timestamp with time zone, title text, url text) returns bytea as $$
      begin
        return digest(feed_id::text || coalesce(publication_time::text, '') || title || url, 'sha256');
      end;
    $$ language plpgsql;

    -- Remove publication_time from obviously bad records
    update items set publication_time = null where publication_time < '1990-01-01';
  `)

	m.AppendMigration("Alter items so digest is calculated by application", `
    drop trigger digest_items on items;
    drop function digest_items();
    drop function digest_item(feed_id integer, publication_time timestamp with time zone, title text, url text);

    with to_delete(id) as (
      select unnest(array_agg(id))
      from items
      group by feed_id, title, url
      having count(*) > 1
      except
      select (array_agg(id))[1]
      from items
      group by feed_id, title, url
      having count(*) > 1
    )
    delete from items
    using to_delete
    where items.id=to_delete.id;

    alter table items drop constraint items_digest_key;
    create unique index items_digest_feed_id_uniq on items (digest, feed_id);

    update items
    set digest = digest(url || title, 'md5');
  `)

	m.AppendMigration("Alter items so URL instead of digest is unique", `
    alter table items drop column digest;

    with to_delete(id) as (
      select unnest(array_agg(id))
      from items
      group by feed_id, url
      having count(*) > 1
      except
      select (array_agg(id))[1]
      from items
      group by feed_id, url
      having count(*) > 1
    )
    delete from items
    using to_delete
    where items.id=to_delete.id;

    create unique index items_feed_id_url_uniq on items (feed_id, url);
  `)

	return m.Migrate()
}
