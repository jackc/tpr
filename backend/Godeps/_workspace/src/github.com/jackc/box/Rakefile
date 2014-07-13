file 'box.go' => 'box.go.erb' do
  sh 'erb box.go.erb | gofmt > box.go'
end

task test: 'box.go' do
  exec 'go test'
end

task default: :test
