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
require 'sass'
require 'erb'
require 'zlib'

CLEAN.include("tmp/js")
CLOBBER.include("public", "tpr")

SRC = FileList["*.go"]
VERSION = File.readlines('main.go').grep(/const version = ".+"/).first[/\d+\.\d+\.\d+/]

directory 'tmp/css'
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

FileList['assets/**/*.scss'].each do |scss_path|
  compiled_css_path = scss_path.sub(/^assets/, 'tmp').sub(/\.scss$/, '.css')

  file compiled_css_path => File.dirname(compiled_css_path)
  file compiled_css_path => scss_path do
    compiled_css = Sass::Engine.for_file(scss_path, {}).render
    File.write compiled_css_path, compiled_css
  end
end

css_files = File.readlines('assets/css/application.css.manifest').map(&:strip).map do |path|
  if path =~ /\.scss$/
    "tmp/css/" + path.sub(/\.scss$/, '.css')
  elsif path =~ /\.css$/
    "assets/css/#{path}"
  else
    raise "Unknown asset type: #{path}"
  end
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
    Zlib::GzipWriter.open(compressed, Zlib::BEST_COMPRESSION) do |gz|
      gz.write File.read(uncompressed)
    end
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
  exec "rerun --dir='.,assets' --pattern='*.{go,manifest,scss,css,js,coffee,erb}' rake server"
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

file "tpr_#{VERSION}.deb" => :build do
  raise 'Must run as root' unless Process.uid == 0

  pkg_dir = "tpr_#{VERSION}"
  FileUtils.rm_rf pkg_dir

  FileUtils.cp_r 'deploy/ubuntu/template', "#{pkg_dir}"

  control_template = File.read("#{pkg_dir}/DEBIAN/control")
  control = ERB.new(control_template).result binding
  File.write "#{pkg_dir}/DEBIAN/control", control

  FileUtils.rm "#{pkg_dir}/usr/bin/.gitignore"
  FileUtils.rm "#{pkg_dir}/usr/share/.gitignore"

  FileUtils.cp 'tpr', "#{pkg_dir}/usr/bin"
  FileUtils.cp_r 'public', "#{pkg_dir}/usr/share/tpr"

  sh "dpkg --build #{pkg_dir}"
  sh "lintian #{pkg_dir}.deb"
end

task deb: "tpr_#{VERSION}.deb"
