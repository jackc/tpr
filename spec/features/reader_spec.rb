require 'spec_helper'

feature 'Reader' do
  scenario 'User marks all items read' do
    user_id = create_user name: 'john', password: 'secret'
    feed_id = create_feed
    DB[:subscriptions].insert user_id: user_id, feed_id: feed_id
    before_item_id = create_item feed_id: feed_id, title: 'First Post', publication_time: Time.local(2014,2,6, 10,34,51)
    DB[:unread_items].insert user_id: user_id, feed_id: feed_id, item_id: before_item_id

    login name: 'john', password: 'secret'

    expect(page).to have_content 'First Post'
    expect(page).to have_content 'February 6th, 2014 at 10:34 am'

    # After user is viewing unread items then add another
    after_item_id = create_item feed_id: feed_id, title: 'Second Post'
    DB[:unread_items].insert user_id: user_id, feed_id: feed_id, item_id: after_item_id

    click_on 'Mark All Read'

    expect(page).to have_content 'Second Post'

    click_on 'Mark All Read'

    # Add another item
    another_item_id = create_item feed_id: feed_id, title: 'Third Post'
    DB[:unread_items].insert user_id: user_id, feed_id: feed_id, item_id: another_item_id

    click_on 'Refresh'

    expect(page).to have_content 'Third Post'
  end

  scenario 'User uses keyboard shortcuts' do
    user_id = create_user name: 'john', password: 'secret'
    feed_id = create_feed
    DB[:subscriptions].insert user_id: user_id, feed_id: feed_id
    item_id = create_item feed_id: feed_id, title: 'First Post'
    DB[:unread_items].insert user_id: user_id, feed_id: feed_id, item_id: item_id
    item_id = create_item feed_id: feed_id, title: 'Second Post'
    DB[:unread_items].insert user_id: user_id, feed_id: feed_id, item_id: item_id

    login name: 'john', password: 'secret'

    # The first item is selected and the second is not
    within('.selected') do
      expect(page).to have_content 'First Post'
      expect(page).to_not have_content 'Second Post'
    end

    # Press "j" to move to next item
    page.find('body').native.send_keys("j")

    # The second item is selected and the first is not
    within('.selected') do
      expect(page).to_not have_content 'First Post'
      expect(page).to have_content 'Second Post'
    end

    visit(current_url)

    # After reloading the page the first post should no longer be visible, but the second should
    expect(page).to_not have_content 'First Post'
    expect(page).to have_content 'Second Post'

    # Can't test for Shift+m :(
    #
    # Failure/Error: page.find('body').native.send_keys([:shift, "m"])
    # Capybara::Poltergeist::Error:
    #   PhantomJS behaviour for key modifiers is currently broken, we will add this in later versions

    # Press Shift+m
    # page.find('body').native.send_keys([:shift, "m"])

    # After marking all read the second post is no longer visible
    # expect(page).to_not have_content 'Second Post'
  end
end
