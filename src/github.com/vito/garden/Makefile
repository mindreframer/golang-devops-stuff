all: protocol skeleton

protocol:
	mkdir -p protocol/
	protoc --gogo_out=protocol/ --proto_path=protobuf/ protobuf/*.proto

skeleton: warden/warden
	rsync -a warden/warden/root/ root/
	cd warden/warden/src && make clean all
	cp warden/warden/src/wsh/wshd root/linux/skeleton/bin
	cp warden/warden/src/wsh/wsh root/linux/skeleton/bin
	cp warden/warden/src/oom/oom root/linux/skeleton/bin
	cp warden/warden/src/iomux/iomux-spawn root/linux/skeleton/bin
	cp warden/warden/src/iomux/iomux-link root/linux/skeleton/bin
	mkdir -p root/bin
	cp warden/warden/src/repquota/repquota root/bin
	go build -o root/linux/skeleton/bin/wshd ./backend/linux_backend/wshd
	go build -o root/linux/skeleton/bin/wsh ./backend/linux_backend/wshd/wsh

warden/warden:
	git submodule update --init --recursive
