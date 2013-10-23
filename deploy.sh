#!/usr/bin/env bash

##
# @author Jay Taylor [@jtaylor]
#
# @date 2013-07-11
#

cd "$(dirname "$0")"

# Verify that `go` and `envdir` (daemontools) dependencies are available.
test -z "$(which go)" && echo 'fatal: no "go" binary found, make sure go-lang is installed and available in a directory in $PATH' 1>&2 && exit 1
test -z "$(which envdir)" && echo 'fatal: no "envdir" binary found, make sure daemontools is installed and and available in $PATH' 1>&2 && exit 1

test ! -d './env' && echo 'fatal: missing "env" configuration directory, see "Compilation" in the README' 1>&2 && exit 1

envdir env go run deploy.go $*
