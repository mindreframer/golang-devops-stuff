FROM ubuntu
MAINTAINER fart

ADD . /opt/gollector/
RUN apt-get update
RUN apt-get install rsyslog curl -y

ENTRYPOINT ["/opt/gollector/docker/dind", "sh", "-c", "rsyslogd -c5 && sleep 5 && /opt/gollector/gollector /opt/gollector/test.json"]
