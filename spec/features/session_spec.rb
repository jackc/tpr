require 'spec_helper'

feature 'Sessions' do
  scenario 'User has invalid session is logged out' do
    create_user name: 'john', password: 'secret'
    login name: 'john', password: 'secret'

    DB[:sessions].delete

    click_on 'Feeds'

    expect(page).to have_content 'User name'
    expect(page).to have_content 'Password'
  end
end
