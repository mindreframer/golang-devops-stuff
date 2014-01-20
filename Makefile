all: gollector gstat

clean:
	rm -f gollector gstat

gollector: gollector.go src/*/*/*.go src/*/*.go
	GOPATH=$(PWD) go build gollector.go

gstat: gstat.go
	GOPATH=$(PWD) go build gstat.go

gollector.tar.gz: gollector gstat
	tar cvzf gollector.tar.gz gollector gstat >/dev/null

dist: all gollector.tar.gz clean

distclean: clean
	rm -f gollector.tar.gz

run: gollector stop
	sh -c './gollector test.json &'

stop: 
	(pkill gollector || exit 0)
