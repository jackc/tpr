require 'spec_helper'

feature 'Account' do
  scenario 'User fails to changes password because of wrong existing password' do
    create_user name: 'john', password: 'secret'
    login name: 'john', password: 'secret'

    click_on 'Account'

    fill_in 'Existing Password', with: 'wrong'
    fill_in 'New Password', with: 'bigsecret'
    fill_in 'Password Confirmation', with: 'bigsecret'
    click_on 'Update'

    accept_alert

    click_on 'Logout'

    login name: 'john', password: 'secret'
  end

  scenario 'User changes email' do
    create_user name: 'john', password: 'secret'
    login name: 'john', password: 'secret'

    click_on 'Account'

    fill_in 'Existing Password', with: 'secret'
    fill_in 'Email', with: 'john@example.com'
    click_on 'Update'

    accept_alert

    click_on 'Feeds'

    click_on 'Account'

    expect(find('#email').value).to eq "john@example.com"
  end
end
