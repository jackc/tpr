module LoginHelper
  def login(name:, password:)
    visit '/#login'

    fill_in 'User name', with: name
    fill_in 'Password', with: password

    click_on 'Login'

    expect(page).to have_no_content 'User name'
  end
end
