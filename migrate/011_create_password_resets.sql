create table password_resets(
  token varchar primary key,
  email varchar not null,
  request_ip inet,
  request_time timestamptz not null,
  user_id integer references users,
  completion_ip inet,
  completion_time timestamptz,
  check(completion_ip is null = completion_time is null)
);

comment on column password_resets.user_id is 'user_id associated with the email at the reset request time';

grant select, insert, update, delete on password_resets to {{.app_user}};

---- create above / drop below ----

drop table password_resets;
