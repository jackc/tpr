create table unread_items(
  user_id integer not null,
  feed_id integer not null,
  item_id integer not null,
  primary key(user_id, feed_id, item_id),
  foreign key (user_id, feed_id) references subscriptions (user_id, feed_id) on delete cascade,
  foreign key (feed_id, item_id) references items (feed_id, id) on delete cascade
);

grant select, insert, update, delete on unread_items to {{.app_user}};

---- create above / drop below ----

drop table unread_items;
