create table subscriptions(
  user_id integer not null references users on delete cascade,
  feed_id integer not null references feeds on delete cascade,
  primary key(user_id, feed_id)
);

create index on subscriptions (feed_id);

grant select, insert, update, delete on subscriptions to {{.app_user}};

---- create above / drop below ----

drop table subscriptions;
