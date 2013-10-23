#!/bin/sh

rsync -acrv --exclude .git --exclude .hg $1 $2
