create table users(
  id serial primary key,
  name varchar(30) not null check(name ~ '\A[a-zA-Z0-9]+\Z'),
  password_digest bytea not null,
  password_salt bytea not null
);

create unique index users_name_unq on users (lower(name));

grant select, insert, update, delete on users to {{.app_user}};

---- create above / drop below ----

drop table users;
