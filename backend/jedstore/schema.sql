create table users(
  id uuid primary key,
  name varchar(30) not null check(name ~ '^[a-zA-Z0-9]+$'),
  password_digest bytea not null,
  password_salt bytea not null
);

create unique index users_name_unq on users (lower(name));


create table feeds(
  id uuid primary key,
  name varchar not null check(name<>''),
  url varchar not null unique check(url<>''),
  last_fetch_time timestamptz,
  etag varchar,
  last_failure varchar,
  last_failure_time timestamptz,
  failure_count integer not null default 0,
  creation_time timestamptz not null default now()
);

create index on feeds (last_fetch_time);


create table items(
  id uuid primary key,
  feed_id uuid not null references feeds on delete cascade,
  publication_time timestamptz,
  title varchar not null,
  url varchar not null,
  creation_time timestamptz not null default now(),
  unique(feed_id, id),
  unique(feed_id, url)
);

create index on items (feed_id);


create table subscriptions(
  user_id uuid not null references users on delete cascade,
  feed_id uuid not null references feeds on delete cascade,
  primary key(user_id, feed_id)
);

create index on subscriptions (feed_id);


create table unread_items(
  user_id uuid not null,
  feed_id uuid not null,
  item_id uuid not null,
  primary key(user_id, feed_id, item_id),
  foreign key (user_id, feed_id) references subscriptions (user_id, feed_id) on delete cascade,
  foreign key (feed_id, item_id) references items (feed_id, id) on delete cascade
);


create table sessions(
  id bytea primary key,
  user_id uuid not null references users on delete cascade,
  start_time timestamptz not null default now()
);


alter table users add column email varchar unique;

create table password_resets(
  token varchar primary key,
  email varchar not null,
  request_ip text,
  request_time timestamptz not null,
  user_id uuid references users,
  completion_ip text,
  completion_time timestamptz,
  check((completion_ip is null) = (completion_time is null))
);
