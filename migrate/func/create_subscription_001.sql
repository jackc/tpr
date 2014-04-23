create function create_subscription(user_id integer, url varchar) returns void as
$$
declare
  feed_id integer;
begin
  loop
    -- try to find existing feed
    select id into feed_id from feeds where feeds.url=create_subscription.url;
    if found then
      exit;
    end if;

    -- if feed was not found then create it
    begin
      insert into feeds(name, url)
        values(create_subscription.url, create_subscription.url)
        returning id into feed_id;

      exit;
    exception when unique_violation then
      -- try again
    end;
  end loop;

  insert into subscriptions(user_id, feed_id) values(user_id, feed_id);
end;
$$
language plpgsql;

grant execute on function create_subscription(integer, varchar) to {{.app_user}};
