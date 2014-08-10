FROM debian:jessie
MAINTAINER Alex Wilson a.wilson@alumni.warwick.ac.uk

RUN apt-get update && \
 apt-get install -qy golang-go git make

RUN mkdir -p /usr/local/go/bin
ENV GOPATH /usr/local/go
ENV GOBIN /usr/local/go/bin
ENV PATH $PATH:$GOBIN

RUN go get github.com/mrwilson/helixdns && \
 go install github.com/mrwilson/helixdns

EXPOSE 53

CMD [ "helixdns", "-port", "53", "-forward", "8.8.8.8:53" ]
