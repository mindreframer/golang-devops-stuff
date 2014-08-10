FROM       arken/gom-base:0.1.0
MAINTAINER Damien Metzler <dmetzler@nuxeo.com>

RUN go get github.com/arkenio/gogeta
WORKDIR /usr/local/go/src/github.com/arkenio/gogeta
RUN git checkout master
RUN gom install
RUN gom test

EXPOSE 7777
ENTRYPOINT ["gogeta", "-etcdAddress", "http://172.17.42.1:4001", "-alsologtostderr=true"]
