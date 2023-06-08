#!/bin/bash

# generates tunascript by invoking ictcc.

cd "$(dirname "$0")"

ictcc --slr \
    -l TunaScript -v 1.0 \
    -d tsi \
    --ir github.com/dekarrin/tunaq/tunascript/syntax.AST \
    --hooks ./syntax \
    ./tunascript.md "$@"

