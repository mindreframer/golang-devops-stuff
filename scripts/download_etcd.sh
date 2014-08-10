#!/bin/sh -x

ETCD_VER=0.4.2

ETCD_DIR=/tmp/etcd
ETCD_ZIP=/tmp/etcd.tar.gz

ETCD_URL=https://github.com/coreos/etcd/releases/download/v${ETCD_VER}/etcd-v${ETCD_VER}-linux-amd64.tar.gz

echo Cleaning up...
rm -rf $ETCD_ZIP $ETCD_DIR

echo Downloading etcd...
curl -L $ETCD_URL -o $ETCD_ZIP

echo Unzipping etcd.tar.gz...
mkdir -p $ETCD_DIR
tar zxvf $ETCD_ZIP -C $ETCD_DIR --strip 1
