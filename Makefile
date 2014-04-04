BATS_DIR := /tmp/bats
ETCD_DIR := /tmp/etcd

install: deps
	@go install ./cmd/hdns

test: deps
	@go test -v

integration-test: install etcd bats
	@$(ETCD_DIR)/etcd > /dev/null &
	@go run ./cmd/hdns/hdns.go &
	@sleep 5
	@$(BATS_DIR)/bin/bats tests/
	@killall -9 etcd hdns

deps:
	@go get github.com/coreos/go-etcd/etcd
	@go get github.com/miekg/dns

etcd:
	@[ -d $(ETCD_DIR) ] || sh ./scripts/download_etcd.sh

bats:
	@[ -d $(BATS_DIR) ] || git clone https://github.com/sstephenson/bats.git $(BATS_DIR)

clean:
	@rm -f $(GOPATH)/bin/hdns
	@rm -rf $(BATS_DIR) $(ETCD_DIR)
