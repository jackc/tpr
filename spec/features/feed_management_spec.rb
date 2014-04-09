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
end
