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

CLOBBER.include("build")

namespace :build do
  task :directory do
    Dir.mkdir("build") unless Dir.exists?("build")
  end

  desc "Build assets"
  task assets: :directory do
    sh "cd frontend; NODE_ENV=production npm run build"
    js_file_name = Dir.glob("build/assets/js/bundle.*.js").first.sub(/^build/, "")
    index_html = File.read "frontend/html/index.html"
    index_html.gsub!("./bundle.js", js_file_name)
    File.write "build/assets/index.html", index_html

    Dir.glob("build/assets/**/*.{js,html}").each do |path|
      sh "zopfli", path
    end
  end

  desc "Build tpr binary"
  task binary: "build/tpr"
end

file "build/tpr" => ["build:directory", *FileList["backend/*.go"]] do |t|
  sh "cd backend; go build -o ../build/tpr github.com/jackc/tpr/backend"
end

desc "Build all"
task build: ["build:assets", "build:binary"]

desc "Run tpr"
task run: "build:binary" do
  puts "Remember to start webpack-dev-server"
  exec "build/tpr server --config tpr.conf --static-url http://localhost:8080"
end

desc "Watch for source changes and rebuild and rerun"
task :rerun do
  exec "react2fs -dir backend rake run"
end

task spec_server: "build:binary" do
  FileUtils.mkdir_p "tmp/spec/server"
  FileUtils.touch "tmp/spec/server/stdout.log"
  FileUtils.touch "tmp/spec/server/stderr.log"
  pid = Process.spawn "build/tpr server --config tpr.test.conf --static-url http://localhost:8080",
    out: "tmp/spec/server/stdout.log",
    err: "tmp/spec/server/stderr.log"
  at_exit { Process.kill "TERM", pid }
end

RSpec::Core::RakeTask.new(:spec)
task spec: :spec_server

desc "Run go tests"
task :test do
  sh "cd backend; go test"
end

task :default => [:test, :spec]
