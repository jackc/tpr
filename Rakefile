begin
  require "bundler"
  Bundler.setup
rescue LoadError
  puts "You must `gem install bundler` and `bundle install` to run rake tasks"
end

require "rake/clean"
require "fileutils"
require "rspec/core/rake_task"
require "erb"
require "md2man/roff/engine"

CLOBBER.include("build")

namespace :build do
  task :directory do
    Dir.mkdir("build") unless Dir.exists?("build")
  end

  desc "Build assets"
  task assets: :directory do
    sh "cd frontend; middleman build"
  end

  desc "Build tpr binary"
  task binary: "build/tpr"

  desc "Build tpr man page"
  task man: "build/tpr.1.gz"
end

file "build/tpr" => ["build:directory", *FileList["backend/*.go"]] do |t|
  sh "cd backend; godep go build -o ../build/tpr github.com/jackc/tpr/backend"
end

file "build/tpr.1.gz" => "man/tpr.md" do
  md_template = File.read("man/tpr.md")
  md = ERB.new(md_template).result binding
  roff = Md2Man::Roff::ENGINE.render(md)

  # Shelling out to gzip instead of doing it in memory because lintian doesn't
  # consider it to have been done at max compression
  File.write "build/tpr.1", roff
  sh "gzip", "-9", "build/tpr.1"
end

desc "Build all"
task build: ["build:assets", "build:binary", "build:man"]

desc "Run tpr"
task run: "build:binary" do
  puts "Remember to start middleman"
  exec "build/tpr server --config tpr.conf --static-url http://localhost:4567"
end

desc "Watch for source changes and rebuild and rerun"
task :rerun do
  exec "react2fs -dir backend rake run"
end

task spec_server: "build:binary" do
  FileUtils.mkdir_p "tmp/spec/server"
  FileUtils.touch "tmp/spec/server/stdout.log"
  FileUtils.touch "tmp/spec/server/stderr.log"
  pid = Process.spawn "build/tpr server --config tpr.test.conf --static-url http://localhost:4567",
    out: "tmp/spec/server/stdout.log",
    err: "tmp/spec/server/stderr.log"
  at_exit { Process.kill "TERM", pid }
end

RSpec::Core::RakeTask.new(:spec)
task spec: :spec_server

desc "Run go tests"
task :test do
  sh "cd backend; godep go test"
end

task :default => [:test, :spec]
