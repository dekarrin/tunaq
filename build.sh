#!/bin/bash

cd "$(dirname "$0")"

env GOFLAGS=-mod=mod go build -o tqi cmd/tqi/main.go
