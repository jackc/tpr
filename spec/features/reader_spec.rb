require 'spec_helper'

feature 'Reader' do
  scenario 'User marks all items read' do
    user_id = create_user name: 'john', password: 'secret'
    feed_id = create_feed
    DB[:subscriptions].insert user_id: user_id, feed_id: feed_id
    before_item_id = create_item feed_id: feed_id, title: 'First Post'
    DB[:unread_items].insert user_id: user_id, feed_id: feed_id, item_id: before_item_id

    visit '/#login'

    fill_in 'User name', with: 'john'
    fill_in 'Password', with: 'secret'

    click_on 'Login'

    expect(page).to have_content 'First Post'

    # After user is viewing unread items then add another
    after_item_id = create_item feed_id: feed_id, title: 'Second Post'
    DB[:unread_items].insert user_id: user_id, feed_id: feed_id, item_id: after_item_id

    click_on 'Mark All Read'

    expect(page).to have_content 'Second Post'
  end
end
