#!/bin/sh -x

ETCD_DIR=/tmp/etcd
ETCD_VER=etcd-v0.2.0-Linux-x86_64
ETCD_URL=https://github.com/coreos/etcd/releases/download/v0.2.0/${ETCD_VER}.tar.gz
ETCD_ZIP=/tmp/etcd.tar.gz

echo Cleaning up...
rm -rf $ETCD_ZIP $ETCD_DIR

echo Downloading etcd...
curl -L $ETCD_URL -o $ETCD_ZIP

echo Unzipping etcd.tar.gz...
mkdir -p $ETCD_DIR
tar zxvf $ETCD_ZIP -C $ETCD_DIR --strip 1
