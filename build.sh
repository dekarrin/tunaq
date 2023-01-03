#!/bin/bash

if ! which go >/dev/null 2>&1
then
  echo "The Go language compiler appears to be uninstalled" >&2
  echo "Install Go from https://go.dev/doc/install and try again" >&2
  exit 1
fi

cd "$(dirname "$0")"

ext=""
[ "$(go env GOOS)" = "windows" ] && ext=".exe"
env GOFLAGS=-mod=mod go build -o tqi$ext cmd/tqi/main.go
