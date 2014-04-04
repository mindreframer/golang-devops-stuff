#!/bin/sh

ETCD_URL=http://localhost:4001/v2/keys/helix/
ETCD_VER=etcd-v0.2.0-Linux-x86_64

set_etcd_record (){
  RECORD=$1
  VALUE=$2
  curl --silent -o /dev/null -XPUT ${ETCD_URL}/${RECORD} -d value="${VALUE}"
}

dig_record (){
  ADDRESS=$1
  TYPE=$2
  dig ${ADDRESS} @localhost -p 9000 ${TYPE} +short
}
