
flags = ""

namespace :build do
  task :deps do
    sh "go get ./..."
  end

  task :host do
    sh "go build #{flags} cmd/tachyon.go"
  end

  task :linux do
    sh "sh -c 'GOOS=linux GOARCH=amd64 go build #{flags} -o tachyon-linux-amd64 cmd/tachyon.go'"
  end

  task :nightly do
    flags = %Q!-ldflags "-X main.Release nightly"!
  end

  task :all => [:host, :linux]
end

namespace :test do
  task :normal do
    sh "go test -v"
  end

  task :package do
    sh "sudo GOPATH=#{ENV['GOPATH']} /usr/bin/go test ./package/apt -v"
  end
end

task :test => ["build:deps", "test:normal", "test:package"]

task :default => :test
