test: clean
	go test -v ./...

cstest:clean
	CASSANDRA=yes go test -v ./backend

coverage: clean
	gocov test -v ./... | gocov report

annotate: clean
	FILENAME=$(shell uuidgen)
	gocov test -v ./... > /tmp/--go-test-server-coverage.json
	gocov annotate /tmp/--go-test-server-coverage.json $(fn)
all:
	go install github.com/mailgun/vulcan # installs library
	go install github.com/mailgun/vulcan/vulcan # and service
deps:
	go get -v -u github.com/axw/gocov
	go install github.com/axw/gocov/gocov
	go get -v -u github.com/golang/glog
	go get -v -u github.com/mailgun/glogutils
	go get -v -u github.com/axw/gocov
	go get -v -u launchpad.net/gocheck
	go get -v -u github.com/mailgun/gocql
clean:
	find . -name flymake_* -delete
run: all
	GOMAXPROCS=4 vulcan -stderrthreshold=INFO -logtostderr=true -c=http://localhost:5000 -b=memory -lb=roundrobin -log_dir=/tmp -logcleanup=24h
csrun: all
	GOMAXPROCS=4 vulcan -stderrthreshold=INFO -logtostderr=true -c=http://localhost:5000 -b=cassandra -lb=roundrobin -csnode=localhost -cskeyspace=vulcan_dev -cscleanup=true -cscleanuptime=19:05 -log_dir=/tmp
sloccount:
	 find . -name "*.go" -print0 | xargs -0 wc -l
