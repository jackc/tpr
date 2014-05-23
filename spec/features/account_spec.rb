require 'spec_helper'

feature 'Account' do
  scenario 'User changes password' do
    create_user name: 'john', password: 'secret'
    login name: 'john', password: 'secret'

    click_on 'Account'

    fill_in 'Existing Password', with: 'secret'
    fill_in 'New Password', with: 'bigsecret'
    fill_in 'Password Confirmation', with: 'bigsecret'
    click_on 'Update'

    page.driver.browser.switch_to.alert.accept

    click_on 'Logout'

    login name: 'john', password: 'bigsecret'
  end
end
