ARCH=m64

.PHONY: pubsubsqlsvc
pubsubsqlsvc: pubsubsqlsvc.cpp
	echo $(ARCH)
	rm -f pubsubsqlsvc
	g++ -Wall -pthread -$(ARCH) pubsubsqlsvc.cpp -o pubsubsqlsvc

