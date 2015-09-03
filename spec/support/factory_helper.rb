require 'ffaker'
require 'scrypt'

module FactoryHelper
  def create_user attrs={}
    defaults = {name: FFaker::Internet.user_name, password: "password"}
    attrs = defaults.merge(attrs)
    attrs[:password_salt] = Sequel::SQL::Blob.new "salt"
    password_digest = SCrypt::Engine.__sc_crypt attrs.delete(:password), attrs[:password_salt], 16384, 8, 1, 32
    attrs[:password_digest] = Sequel::SQL::Blob.new password_digest
    DB[:users].insert attrs
  end

  def create_feed attrs={}
    defaults = {
      name: FFaker::Lorem.sentence,
      url: "http://localhost/#{FFaker::Internet.domain_word}/#{FFaker::Internet.domain_word}"
    }
    attrs = defaults.merge(attrs)
    DB[:feeds].insert attrs
  end

  def create_item attrs={}
    attrs[:feed_id] ||= create_feed # not part of defaults hash so we don't create feed unless we need to
    defaults = {
      title: FFaker::Lorem.sentence,
      url: FFaker::Internet.http_url
    }
    attrs = defaults.merge(attrs)
    DB[:items].insert attrs
  end
end
