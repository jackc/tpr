require 'capybara/rspec'
require 'capybara/poltergeist'
require 'sequel'
require 'pry'
require 'yaml'

Dir["#{File.dirname(__FILE__)}/support/**/*.rb"].each {|f| require f}

config = YAML.load File.read('config.test.yml')
host = config['database']['socket'].sub(/\/[^\/]+$/, '')

DB = Sequel.postgres host: host, database: config['database']['database']

Capybara.default_driver = :poltergeist
Capybara.app_host = "http://#{config['address']}:#{config['port']}"

RSpec.configure do |config|
  config.before(:each) do
    clean_database
    visit '/'
  end
end