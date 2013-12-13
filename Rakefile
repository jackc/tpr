begin
  require 'bundler'
  Bundler.setup
rescue LoadError
  puts 'You must `gem install bundler` and `bundle install` to run rake tasks'
end

require 'rake/clean'
require 'fileutils'
require 'rspec/core/rake_task'
require 'coffee_script'
require 'erb'

CLEAN.include("views.go", "tmp/js")
CLOBBER.include("tpr")

SRC = FileList["*.go"]

directory 'tmp/js'
directory 'tmp/js/collections'
directory 'tmp/js/models'
directory 'tmp/js/services'
directory 'tmp/js/views'
directory 'public'
directory 'public/js'
directory 'public/css'

FileList['assets/**/*.coffee'].each do |coffee_path|
  compiled_js_path = coffee_path.sub(/^assets/, 'tmp').sub(/\.coffee$/, '.js')

  file compiled_js_path => File.dirname(compiled_js_path)
  file compiled_js_path => coffee_path do
    compiled_js = CoffeeScript.compile File.read(coffee_path)
    File.write compiled_js_path, compiled_js
  end
end

js_files = File.readlines('assets/js/application.js.manifest').map(&:strip).map do |path|
  if path =~ /\.coffee$/
    "tmp/js/" + path.sub(/\.coffee$/, '.js')
  elsif path =~ /\.js$/
    "assets/js/" + path
  else
    raise "Unknown asset type: #{path}"
  end
end

file 'public/js/application.js' => 'public/js'
file 'public/js/application.js' => 'assets/js/application.js.manifest'
file 'public/js/application.js' => js_files do
  File.open 'public/js/application.js', 'w' do |output|
    js_files.each do |source_path|
      puts source_path
      output.puts File.read source_path
    end
  end
end

css_files = File.readlines('assets/css/application.css.manifest').map(&:strip).map do |path|
  "assets/css/#{path}"
end

file 'public/css/application.css' => 'public/css'
file 'public/css/application.css' => 'assets/css/application.css.manifest'
file 'public/css/application.css' => css_files do
  File.open 'public/css/application.css', 'w' do |output|
    css_files.each do |source_path|
      output.puts File.read source_path
    end
  end
end

file 'public/index.html' => 'public'
file 'public/index.html' => 'assets/html/index.html.erb' do
  File.write('public/index.html', File.read('assets/html/index.html.erb'))
end

[
  'public/js/application.js',
  'public/css/application.css',
].each do |uncompressed|
  compressed = uncompressed + '.gz'

  file compressed => uncompressed do
    sh 'gzip', '--keep', '--best', '--force', uncompressed
  end
end

file 'tpr' => SRC do |t|
  sh 'go build'
end

desc 'Build tpr'
task build: ['tpr', 'public/index.html', 'public/js/application.js', 'public/css/application.css', 'public/js/application.js.gz', 'public/css/application.css.gz']

desc 'Run tpr server'
task server: :build do
  exec './tpr'
end

desc "Run tpr server and restart when source change"
task :rerun do
  exec "rerun --dir='.,assets' --pattern='*.{go,manifest,css,js,coffee,erb}' rake server"
end

task spec_server: :build do
  FileUtils.mkdir_p 'tmp/spec/server'
  FileUtils.touch 'tmp/spec/server/stdout.log'
  FileUtils.touch 'tmp/spec/server/stderr.log'
  pid = Process.spawn './tpr -config=config.test.yml',
    out: 'tmp/spec/server/stdout.log',
    err: 'tmp/spec/server/stderr.log'
  at_exit { Process.kill 'TERM', pid }
end

RSpec::Core::RakeTask.new(:spec)
task spec: :spec_server

task :default => :spec
