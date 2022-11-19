grant truncate on users to {{.app_user}};
grant truncate on feeds to {{.app_user}};
grant truncate on items to {{.app_user}};
grant truncate on unread_items to {{.app_user}};
grant truncate on sessions to {{.app_user}};
grant truncate on subscriptions to {{.app_user}};

---- create above / drop below ----

revoke truncate on users to {{.app_user}};
revoke truncate on feeds to {{.app_user}};
revoke truncate on items to {{.app_user}};
revoke truncate on unread_items to {{.app_user}};
revoke truncate on sessions to {{.app_user}};
revoke truncate on subscriptions to {{.app_user}};
