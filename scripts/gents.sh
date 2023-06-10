#!/bin/bash

# Generates TunaScript frontend by invoking ictcc.

# --------------------------------------------------

# "is not a function that exists in TunaScript" DURING VALIDATION:
#
# Until Ictiobus v1.1.0, it's impossible to tell when a hook is running under
# validation. Validation simulation will produce semantically invalid function
# calls. For now, disable the arity and existence check when doing simulation.

# CONNECT: NETWORK IS UNREACHABLE:
#
# This script often blows up with errors that end with "connect: network is
# unreachable" during the simulation binary and diagnostics binary generation
# phase (specifically when running go tidy on the updated go.mod). This is
# likely due to some sort of anti-flooding, either in the Go shellout env or
# from the go proxy server itself. Either way, the current workaround until it's
# fixed in Ictiobus is:
#
# 1. Wait ~15 seconds, and try again. The initial goal is to get at least
# *simulation* to build so that the validation tests can run. Repeat this until
# simulation runs successfully (or has errors, in which case go fix the errors
# and then come back).
# 2. If after a successful simulation build, it promptly fails on diagnostics
# bin generation, wait ~15 seconds, then re-run this script with --sim-off. This
# will skip simulation and run go tidy *only* on the diag bin, which is fine
# because simulation already occured. Repeat this step until the diagnostics
# bin builds.

cd "$(dirname "$0")"/..

ictcc --slr \
    -l TunaScript -v 1.0 \
    -d tsi \
    --ir github.com/dekarrin/tunaq/tunascript/syntax.AST \
    --hooks ./tunascript/syntax \
    --dest ./tunascript/fe \
    tunascript/tunascript.md "$@"

