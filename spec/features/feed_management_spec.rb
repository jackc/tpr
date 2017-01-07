require 'spec_helper'

feature 'Feed Management' do
  scenario 'User subscribes to a feed' do
    create_user name: 'john', password: 'secret'
    login name: 'john', password: 'secret'

    click_on 'Feeds'

    fill_in 'Feed URL', with: 'http://localhost:1234'
    click_on 'Subscribe'

    within '.feeds > ul' do
      expect(page).to have_content 'http://localhost:1234'
    end
  end

  scenario 'User imports OPML file' do
    create_user name: 'john', password: 'secret'
    login name: 'john', password: 'secret'

    click_on 'Feeds'

    attach_file 'OPML File', File.expand_path('spec/fixtures/opml.xml')
    click_on 'Import'

    accept_alert

    within '.feeds > ul' do
      expect(page).to have_content 'http://localhost/rss'
      expect(page).to have_content 'http://localhost/other/rss'
    end
  end
end
