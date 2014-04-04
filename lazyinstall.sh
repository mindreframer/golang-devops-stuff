#!/bin/sh -e
set -e # yeah, won't ignore errors

DEST=~/bin/ostent # change if you wish. the directory (~/bin) must be writable for ostent to self-update
URL="https://OSTROST.COM/ostent/releases/latest/$(uname -sm)/ostent"

curl -sSL --create-dirs -o "$DEST" "$URL"
chmod +x "$DEST"

echo All good, installed into: "$DEST"
