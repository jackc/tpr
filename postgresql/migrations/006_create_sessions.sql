create unlogged table sessions(
  id bytea primary key,
  user_id integer not null references users on delete cascade,
  start_time timestamp with time zone not null default now()
);

grant select, insert, update, delete on sessions to {{.app_user}};

---- create above / drop below ----

drop table sessions;
