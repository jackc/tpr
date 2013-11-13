require 'spec_helper'

feature 'Registration' do
  scenario 'Registering a new user' do
    visit '/#login'

    click_on 'Create an account'

    fill_in 'User name', with: 'joe'
    fill_in 'Password', with: 'bigsecret'
    fill_in 'Password Confirmation', with: 'bigsecret'

    click_on 'Register'

    expect(page).to have_content 'Logout'
    expect(DB[:users].count).to eq 1
  end
end
