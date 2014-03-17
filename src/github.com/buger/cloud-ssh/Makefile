all: build-macosx build-x86 build-x64

build-x64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build
	tar -czf cloud_ssh_x64.tar.gz cloud-ssh
	rm cloud-ssh

build-x86:
	GOOS=linux GOARCH=386 CGO_ENABLED=0 go build
	tar -czf cloud_ssh_x86.tar.gz cloud-ssh
	rm cloud-ssh

build-macosx:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build
	tar -czf cloud_ssh_macosx.tar.gz cloud-ssh
	rm cloud-ssh