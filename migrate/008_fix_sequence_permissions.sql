grant usage on sequence items_id_seq to {{.app_user}};
grant usage on sequence feeds_id_seq to {{.app_user}};
grant usage on sequence users_id_seq to {{.app_user}};

---- create above / drop below ----

revoke usage on sequence items_id_seq to {{.app_user}};
revoke usage on sequence feeds_id_seq to {{.app_user}};
revoke usage on sequence users_id_seq to {{.app_user}};
