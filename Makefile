test: clean
	go test -v ./...

testinstall:
	go test -i github.com/mailgun/vulcan/client
	go test -i github.com/mailgun/vulcan/command
	go test -i github.com/mailgun/vulcan/ratelimit
	go test -i github.com/mailgun/vulcan/control/js

cstest:clean
	CASSANDRA=yes go test -v ./backend

cmdtest:clean
	go test -v ./command

jstest:clean
	go test -v ./control/js

ratetest:clean
	go test -v ./ratelimit

proxytest:clean
	go test -v .

all:
	go install github.com/mailgun/vulcan # installs library
	go install github.com/mailgun/vulcan/vulcan # and service

deps:
	go get -v -u code.google.com/p/go.tools/cover
	go get -v -u github.com/axw/gocov
	go install github.com/axw/gocov/gocov
	go get -v -u github.com/golang/glog
	go get -v -u github.com/mailgun/glogutils
	go get -v -u github.com/axw/gocov
	go get -v -u launchpad.net/gocheck
	go get -v -u github.com/mailgun/gocql
	go get -v -u github.com/robertkrimen/otto
	go get -v -u github.com/coreos/go-etcd/etcd
	go get -v -u github.com/mailgun/minheap
	go get -v -u github.com/rcrowley/go-metrics
	go get -v -u github.com/rackspace/gophercloud

clean:
	find . -name flymake_* -delete

run: all
	vulcan -stderrthreshold=INFO -logtostderr=true -js=./examples/hello.js -b=memory -lb=roundrobin -log_dir=/tmp -logcleanup=24h

csrun: all
	vulcan -stderrthreshold=INFO -logtostderr=true -b=cassandra -lb=roundrobin -csnode=localhost -cskeyspace=vulcan_dev -cscleanup=true -cscleanuptime=19:05 -log_dir=/tmp

run-discover: all
	vulcan -stderrthreshold=INFO -logtostderr=true -js=./examples/discover.js -b=memory -lb=roundrobin -log_dir=/tmp -logcleanup=24h -etcd=http://127.0.0.1:4001

sloccount:
	 find . -name "*.go" -print0 | xargs -0 wc -l
