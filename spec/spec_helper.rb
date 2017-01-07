require 'capybara/rspec'
require 'sequel'
require 'pry'
require 'inifile'

Dir["#{File.dirname(__FILE__)}/support/**/*.rb"].each {|f| require f}

config = IniFile.load 'tpr.test.conf'
host = if config['database']['socket']
  config['database']['socket']
else
  config['database']['host']
end

DB = Sequel.postgres host: host,
  password: config['database']['password'],
  user: config['database']['user'],
  database: config['database']['database']

Capybara.register_driver :selenium do |app|
  Capybara::Selenium::Driver.new(app, :browser => :chrome)
end

Capybara.default_driver = :selenium
Capybara.app_host = "http://#{config['server']['address']}:#{config['server']['port']}"

RSpec.configure do |config|
  config.include FactoryHelper
  config.include LoginHelper
  config.include AlertHelper

  config.before(:each) do
    clean_database
    visit '/'
  end
end
