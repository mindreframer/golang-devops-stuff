get: get-code godep

get-code:
	go get $(GO_EXTRAFLAGS) -u -d -t ./...

godep:
	go get $(GO_EXTRAFLAGS) github.com/tools/godep
	godep restore ./...

test:
	go clean $(GO_EXTRAFLAGS) ./...
	go test $(GO_EXTRAFLAGS) ./...

doc:
	@cd docs && make html

run:
	@godep go run webserver/main.go -config ./etc/gandalf.conf
