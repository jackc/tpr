require 'spec_helper'

feature 'Registration' do
  scenario 'Registering a new user' do
    visit '/#login'

    click_on 'Create an account'

    fill_in 'User name', with: 'joe'
    fill_in 'Email (optional)', with: 'joe@example.com'
    fill_in 'Password', with: 'bigsecret'
    fill_in 'Password Confirmation', with: 'bigsecret'

    click_on 'Register'

    expect(page).to have_content 'No unread items'
    expect(DB[:users].count).to eq 1
    user = DB[:users].first
    expect(user[:name]).to eq 'joe'
    expect(user[:email]).to eq 'joe@example.com'
  end

  scenario 'Registering a new user without an email' do
    visit '/#login'

    click_on 'Create an account'

    fill_in 'User name', with: 'joe'
    fill_in 'Password', with: 'bigsecret'
    fill_in 'Password Confirmation', with: 'bigsecret'

    click_on 'Register'

    expect(page).to have_content 'No unread items'
    expect(DB[:users].count).to eq 1
    user = DB[:users].first
    expect(user[:name]).to eq 'joe'
    expect(user[:email]).to eq nil
  end
end
