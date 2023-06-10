#!/bin/bash

# this file builds distributions for 3 major operating systems.

cd "$(dirname "$0")"/..

# fail immediately on first error
set -eo pipefail

if [ -z "$PLATFORMS" ]
then
  PLATFORMS="
darwin/amd64
windows/amd64
linux/amd64"
fi


# only do skip tests if tests have already been done.
[ "$1" = "--skip-tests" ] && skip_tests=1

BINARY_NAME="tqi"
MAIN_SOURCE_FILE="cmd/tqi/main.go"
ARCHIVE_NAME="tunaquest"

tar_cmd=tar
if [ "$(uname -s)" = "Darwin" ]
then
	if tar --version | grep bsdtar >/dev/null 2>&1
	then
		if ! gtar --version >/dev/null 2>&1
		then
			echo "You appear to be running on a mac where 'tar' is BSD tar." >&2
			echo "This will cause issues due to its adding of non-standard headers." >&2
			echo "" >&2
			echo "Please install GNU tar and make it available as 'gtar' with:" >&2
			echo "  brew install gnu-tar" >&2
			echo "And then try again" >&2
			exit 1
		else
			tar_cmd=gtar
		fi
	fi
fi


version="$(go run cmd/tqi/main.go -version)"
if [ -z "$version" ]
then
	echo "could not get version number; abort" >&2
	exit 1
fi

echo "Creating distributions for $ARCHIVE_NAME version $version"

rm -rf "$BINARY_NAME" "$BINARY_NAME.exe"
rm -rf "source.tar.gz"
rm -rf *-source/

if [ -z "$skip_tests" ]
then
  go clean
  go get ./... || { echo "could not install dependencies; abort" >&2 ; exit 1 ; }
  echo "Running unit tests..."
  if go test -timeout 30s ./...
  then
    echo "Unit tests passed"
  else
    echo "unit tests failed; refusing to create distributions for binary that fails tests" >&2
    exit 1
  fi
else
  echo "Skipping tests due to --skip-tests flag; make sure they are executed elsewhere"
fi

source_dir="$ARCHIVE_NAME-$version-source"
git archive --format=tar --prefix="$source_dir/" HEAD | tar xf -
"$tar_cmd" czf "source.tar.gz" "$source_dir"
rm -rf "$source_dir"

for p in $PLATFORMS
do
  current_os="${p%/*}"
  current_arch="${p#*/}"
  echo "Building for $current_os on $current_arch..."

  dist_bin_name="$BINARY_NAME"
  if [ "$current_os" = "windows" ]
  then
    dist_bin_name="${BINARY_NAME}.exe"
  fi

  go clean
  env CGO_ENABLED=0 GOOS="$current_os" GOARCH="$current_arch" go build -o "$dist_bin_name" "$MAIN_SOURCE_FILE" || { echo "build failed; abort" >&2 ; exit 1 ; }

  dist_versioned_name="$ARCHIVE_NAME-$version-$current_os-$current_arch"
  dist_latest_name="$ARCHIVE_NAME-latest-$current_os-$current_arch"

  distfolder="$dist_versioned_name"
  rm -rf "$distfolder" "$dist_latest_name.tar.gz" "$dist_versioned_name.tar.gz"
  mkdir "$distfolder"
  mkdir "$distfolder/docs"
  mkdir "$distfolder/world"
  cp README.md source.tar.gz world.tqw "$distfolder"
  cp docs/tunascript.md "$distfolder/docs/"
  cp docs/tqwformat.md "$distfolder/docs/"
  cp -R world/* "$distfolder/world/"
  
  if [ "$current_os" != "windows" ]
  then
    # no need to set executable bit on windows
    chmod +x "$dist_bin_name"
  fi
  mv $dist_bin_name "$distfolder/"
  $tar_cmd czf "$dist_versioned_name.tar.gz" "$distfolder"
  rm -rf "$distfolder"

  echo "$dist_versioned_name.tar.gz"
  cp "$dist_versioned_name.tar.gz" "$dist_latest_name.tar.gz"
  echo "$dist_latest_name.tar.gz"
done

rm -rf source.tar.gz
