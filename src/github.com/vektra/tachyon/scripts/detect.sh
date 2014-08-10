#!/bin/bash

arch=`uname -m`

case $arch in
  x86_64 )
    arch="amd64" ;;
  486 | 586 | 686 )
    arch="386" ;;
  * )
    echo "Unsupported arch: $arch"
    exit 1
    ;;
esac

os=`uname`

case $os in
  Darwin )
    os="darwin"
    ;;
  Linux )
    os="linux"
    ;;
  * )
    echo "Unsupported os: $os"
    exit 1
esac

echo "tachyon-$os-$arch"
