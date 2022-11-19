create table items(
  id serial primary key,
  feed_id integer not null references feeds on delete cascade,
  publication_time timestamp with time zone,
  title varchar not null,
  url varchar not null,
  creation_time timestamp with time zone not null default now(),
  unique(feed_id, id),
  unique(feed_id, url)
);

create index on items (feed_id);

grant select, insert, update, delete on items to {{.app_user}};

---- create above / drop below ----

drop table items;
