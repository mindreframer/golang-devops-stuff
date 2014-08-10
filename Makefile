all: goget gollector gstat gollector-graphite

goget:
	GOPATH=${PWD} go get -u -d github.com/gollector/gollector_metrics

clean:
	rm -f gollector gstat gollector-graphite

gollector-graphite: goget gollector-graphite.go
	GOPATH=${PWD} go build gollector-graphite.go

gollector: goget gollector.go src/*/*/*.go src/*/*.go
	GOPATH=${PWD} go build gollector.go

gstat: goget gstat.go
	GOPATH=${PWD} go build gstat.go

gollector.tar.gz: gollector gstat
	tar cvzf gollector.tar.gz gollector gstat gollector-graphite >/dev/null

dist: all gollector.tar.gz clean

distclean: clean
	rm -f gollector.tar.gz

run: gollector stop
	sh -c './gollector test.json &'

stop: 
	(pkill gollector || exit 0)
