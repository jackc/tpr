require 'ffaker'
require 'scrypt'

module FactoryHelper
  def create_user attrs={}
    defaults = {name: Faker::Internet.user_name, password: "password"}
    attrs = defaults.merge(attrs)
    attrs[:password_salt] = Sequel::SQL::Blob.new "salt"
    password_digest = SCrypt::Engine.__sc_crypt attrs.delete(:password), attrs[:password_salt], 16384, 8, 1, 32
    attrs[:password_digest] = Sequel::SQL::Blob.new password_digest
    DB[:users].insert attrs
  end
end
