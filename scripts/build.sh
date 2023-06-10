#!/bin/bash

set -eo pipefail

if ! which go >/dev/null 2>&1
then
  echo "The Go language compiler appears to be uninstalled" >&2
  echo "Install Go from https://go.dev/doc/install and try again" >&2
  exit 1
fi

cd "$(dirname "$0")"/..

ext=
for_windows=
[ "$(go env GOOS)" = "windows" ] && for_windows=1 && ext=".exe"

env GOFLAGS=-mod=mod go build -o tqi$ext cmd/tqi/main.go

if [ -n "$for_windows" ]
then
  printf '%s\r\n' '@ECHO OFF' > tqi.bat
  printf '%s\r\n' 'TITLE TunaQuest' >> tqi.bat
  printf '%s\r\n' '%~dp0tqi.exe %*' >> tqi.bat
  printf '%s\r\n' 'PAUSE' >> tqi.bat
  printf '%s\r\n' 'EXIT' >> tqi.bat

  cat <<- 'EOF' > "tqi.sh"
  #!/bin/bash

  cd "$(dirname "$0")"

  if [ "$TERM_PROGRAM" = "mintty" ]
  then
    echo "You are running this from git-bash or other mintty console" >&2
    echo "Readline implementation fails in mintty, opening in new windows console..." >&2

    # windows command here:
    start //wait tqi.bat "$@"
  else
    ./tqi "$@"
  fi
EOF

  echo "Built tqi.exe and tqi.sh for Windows."
  echo "Launch with tqi.sh to enable shell detection. Calling tqi directly may fail."
fi
