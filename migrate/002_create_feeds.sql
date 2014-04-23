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

grant select, insert, update, delete on feeds to {{.app_user}};

---- create above / drop below ----

drop table feeds;
