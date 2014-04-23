{{ template "func/create_subscription_001.sql" . }}

---- create above / drop below ----

drop function create_subscription(integer, varchar);
