begin
  require 'bundler'
  Bundler.setup
rescue LoadError
  puts 'You must `gem install bundler` and `bundle install` to run rake tasks'
end

require 'rake/clean'
require 'fileutils'
require 'rspec/core/rake_task'

CLEAN.include("views.go")
CLOBBER.include("reader")

SRC = FileList["*.go"]
VIEWS = FileList["views/*.gst"]

file 'views.go' => VIEWS do |t|
  sh "gst #{VIEWS.join(' ')} | gofmt > #{t.name}"
end

file 'reader' => [*SRC, 'views.go'] do |t|
  sh 'go build'
end

desc 'Build reader'
task build: 'reader'

desc 'Run reader server'
task server: 'reader' do
  exec './reader'
end

task spec_server: :build do
  FileUtils.mkdir_p 'tmp/spec/server'
  FileUtils.touch 'tmp/spec/server/stdout.log'
  FileUtils.touch 'tmp/spec/server/stderr.log'
  pid = Process.spawn './reader -config=config.test.yml',
    out: 'tmp/spec/server/stdout.log',
    err: 'tmp/spec/server/stderr.log'
  at_exit { Process.kill 'TERM', pid }
end

RSpec::Core::RakeTask.new(:spec)
task spec: :spec_server

task :default => :spec
