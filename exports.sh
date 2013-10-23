#!/usr/bin/env bash

work=$(python -c 'import os, sys;print os.path.abspath(os.path.dirname(os.path.realpath(sys.argv[1])))' $0)
export GOPATH=$work/

if [ -d /data/tmp ]; then
    export TMPDIR=/data/tmp
fi
