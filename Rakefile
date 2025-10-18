begin
  require "bundler"
  Bundler.setup
rescue LoadError
  puts "You must `gem install bundler` and `bundle install` to run rake tasks"
end

require "rake/clean"
require "fileutils"
require "erb"

CLOBBER.include("build")

directory "tmp/test"

namespace :build do
  task :directory do
    Dir.mkdir("build") unless Dir.exist?("build")
  end

  desc "Build assets"
  task assets: :directory do
    sh "npx vite --outDir ../build/assets build frontend"
    Dir.glob("build/assets/**/*.{js,html}").each do |path|
      sh "zopfli", path
    end
  end

  desc "Build tpr binary"
  task binary: ["build/tpr"]
end

file "build/tpr" => ["build:directory", *FileList["backend/*.go"]] do |t|
  sh "go build -o build/tpr"
end

file "build/tpr-linux" => ["build:directory", *FileList["backend/*.go"]] do |t|
  sh "cd backend; GOOS=linux GOARCH=amd64 go build -o ../build/tpr-linux github.com/jackc/tpr/backend"
end

desc "Build all"
task build: ["build:assets", "build:binary", "build/tpr-linux"]

desc "Run tpr"
task run: "build:binary" do
  puts "Remember to start vite dev server"
  exec "build/tpr server --config tpr.conf --static-url http://localhost:5173"
end

desc "Watch for source changes and rebuild and rerun"
task :rerun do
  exec %q[watchexec --project-origin . -r -f Rakefile -f main.go -f "backend/**" -- rake run]
end

file "tmp/test/.databases-prepared" => FileList["postgresql/**/*.sql", "test/testdata/*.sql"] do
  sh "psql -f test/setup_test_databases.sql > /dev/null"
  sh "touch tmp/test/.databases-prepared"
end

desc "Perform all preparation necessary to run tests"
task "test:prepare" => ["tmp/test", "tmp/test/.databases-prepared"]

desc "Run go tests"
task test: ["test:prepare"] do
  sh "go test ./..."
end

task :default => :test
