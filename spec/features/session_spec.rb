require 'spec_helper'

feature 'Sessions' do
  scenario 'User has invalid session can still log out' do
    create_user name: 'john', password: 'secret'

    visit '/#login'

    fill_in 'User name', with: 'john'
    fill_in 'Password', with: 'secret'

    click_on 'Login'

    expect(page).to have_content 'Logout'

    DB[:sessions].delete

    click_on 'Logout'

    expect(page).to have_content 'User name'
    expect(page).to have_content 'Password'
  end
end
