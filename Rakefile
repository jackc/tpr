begin
  require "bundler"
  Bundler.setup
rescue LoadError
  puts "You must `gem install bundler` and `bundle install` to run rake tasks"
end

require "rake/clean"
require "fileutils"
require "rspec/core/rake_task"

CLOBBER.include("build")

VERSION = File.readlines("backend/main.go").grep(/const version = ".+"/).first[/\d+\.\d+\.\d+/]

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
end

file "build/tpr" => ["build:directory", *FileList["backend/*.go"]] do |t|
  sh "go build -o build/tpr github.com/JackC/tpr/backend"
end

desc "Build assets and binary"
task build: ["build:assets", "build:binary"]

desc "Run tpr"
task run: "build:binary" do
  puts "Remember to start middleman"
  exec "build/tpr -config config.yml -static-url http://localhost:4567"
end

desc "Watch for source changes and rebuild and rerun"
task :rerun do
  exec "rerun -d backend -p '**/*.*' rake run"
end

task spec_server: "build:binary" do
  FileUtils.mkdir_p "tmp/spec/server"
  FileUtils.touch "tmp/spec/server/stdout.log"
  FileUtils.touch "tmp/spec/server/stderr.log"
  pid = Process.spawn "build/tpr -config=config.test.yml",
    out: "tmp/spec/server/stdout.log",
    err: "tmp/spec/server/stderr.log"
  at_exit { Process.kill "TERM", pid }
end

RSpec::Core::RakeTask.new(:spec)
task spec: :spec_server

task :default => :spec

file "tpr_#{VERSION}.deb" => :build do
  raise "Must run as root" unless Process.uid == 0

  pkg_dir = "tpr_#{VERSION}"
  FileUtils.rm_rf pkg_dir

  FileUtils.cp_r "deploy/ubuntu/template", "#{pkg_dir}"

  control_template = File.read("#{pkg_dir}/DEBIAN/control")
  control = ERB.new(control_template).result binding
  File.write "#{pkg_dir}/DEBIAN/control", control

  FileUtils.rm "#{pkg_dir}/usr/bin/.gitignore"
  FileUtils.rm "#{pkg_dir}/usr/share/.gitignore"

  FileUtils.cp "build/tpr", "#{pkg_dir}/usr/bin"
  FileUtils.cp_r "build/assets", "#{pkg_dir}/usr/share/tpr"

  sh "dpkg --build #{pkg_dir}"
  sh "lintian #{pkg_dir}.deb"
end

task deb: "tpr_#{VERSION}.deb"
