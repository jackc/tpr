module AlertHelper
  def accept_alert
    retries = 0

    begin
      page.driver.browser.switch_to.alert.accept
    rescue Selenium::WebDriver::Error::NoSuchAlertError
      sleep(0.2)
      retries += 1
      retry if retries < 3
    end
  end
end
