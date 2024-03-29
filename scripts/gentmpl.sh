#!/bin/bash

# Generates the TQ template parser frontend by invoking ictcc.

# --------------------------------------------------

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


# Right now we've disabled simulation entirely because it hangs. Should be fixed
# in Ictiobus 1.1.0, but for now, just don't generate it.

ictcc --slr \
    -l 'TunaQuest Template' -v 1.0 \
    -d tte \
    --sim-off \
    --ir github.com/dekarrin/tunaq/tunascript/syntax.Template \
    --hooks ./tunascript/syntax --hooks-table TmplHooksTable \
    --dest ./tunascript/fetmpl --pkg fetmpl \
    tunascript/expansion.md "$@"
