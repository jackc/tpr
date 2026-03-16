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

# Build matrix: all OS/arch combinations
BUILD_TARGETS = [
  {os: "linux", arch: "amd64"},
  {os: "linux", arch: "arm64"},
  {os: "darwin", arch: "amd64"},
  {os: "darwin", arch: "arm64"},
].freeze

GO_SOURCES = FileList["Rakefile", "*.go", "go.*", "**/*.go"].exclude(/_test.go$/)
ASSET_SOURCES = FileList["src/**/*", "package.json", "vite.config.js", "svelte.config.js"]

directory "tmp/test"

# This task is for development convenience - it builds the binary for the current platform.
file "build/tpr" => GO_SOURCES do |t|
  mkdir_p "build"
  sh "go build -o build/tpr"
end

# Asset build with tracking file for dependency management
file "build/assets/.built" => ASSET_SOURCES do
  sh "npm run build"
  touch "build/assets/.built"
end

# Generate file tasks for each target
BUILD_TARGETS.each do |target|
  dir = "build/#{target[:os]}_#{target[:arch]}"
  binary = "#{dir}/tpr"
  assets_dir = "#{dir}/assets"

  # Binary depends on Go sources
  file binary => GO_SOURCES do |t|
    mkdir_p dir
    sh "GOOS=#{target[:os]} GOARCH=#{target[:arch]} go build -o #{t.name}"
  end

  # Assets copy depends on asset build
  file "#{assets_dir}/.copied" => "build/assets/.built" do |t|
    rm_rf assets_dir
    cp_r "build/assets", assets_dir
    touch t.name
  end

  # VERSION file with git commit hash
  version_file = "#{dir}/VERSION"
  task version_file do |t|
    mkdir_p dir
    commit = `git rev-parse HEAD`.chomp
    dirty = `git status --porcelain`.strip.empty? ? "" : "-dirty"
    File.write(t.name, "#{commit}#{dirty}\n")
  end

  # Convenience task for full build directory
  desc "Build artifact for #{target[:os]}/#{target[:arch]}"
  task dir => [binary, "#{assets_dir}/.copied", version_file]

  # Tarball of the release directory
  tarball = "#{dir}.tar.gz"
  file tarball => dir do |t|
    sh "tar -czf #{t.name} -C #{dir} ."
  end
end

namespace :build do
  desc "Build assets"
  task assets: "build/assets/.built"

  desc "Build tpr binary"
  task binary: ["build/tpr"]
end

desc "Build all"
task build: ["build/tpr", "build/assets/.built"]

desc "Run tpr"
task run: "build:binary" do
  puts "Remember to start vite dev server"
  exec "build/tpr server"
end

desc "Watch for source changes and rebuild and rerun"
task :rerun do
  exec %q[watchexec --project-origin . -r -f Rakefile -f "*.go" -f "backend/**" -- rake run]
end

file "tmp/test/.databases-prepared" => FileList["postgresql/**/*.sql", "test/testdata/*.sql"] do
  sh "psql -f test/setup_test_databases.sql > /dev/null"
  sh "touch tmp/test/.databases-prepared"
end

desc "Perform all preparation necessary to run tests"
task "test:prepare" => ["tmp/test", "tmp/test/.databases-prepared"]

desc "Run Go tests"
task "test:backend" => ["test:prepare"] do
  sh "go test ./..."
end

desc "Run Playwright browser tests"
task "test:browser" => ["test:prepare"] do
  sh "npm run test:e2e"
end

desc "Run all tests"
task test: ["test:backend", "test:browser"]

task default: :test
