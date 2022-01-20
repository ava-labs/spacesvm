#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if ! [[ "$0" =~ scripts/build.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

# Load the constants
# Set the PATHS
GOPATH="$(go env GOPATH)"

# Set default binary directory location
binary_directory="$GOPATH/src/github.com/ava-labs/avalanchego/build/plugins"
name="sqja3uK17MJxfC7AN8nGadBw9JK5BcrsNwNynsqP5Gih8M5Bm"

if [[ $# -eq 1 ]]; then
    binary_directory=$1
elif [[ $# -eq 2 ]]; then
    binary_directory=$1
    name=$2
elif [[ $# -ne 0 ]]; then
    echo "Invalid arguments to build spacesvm. Requires either no arguments (default) or one arguments to specify binary location."
    exit 1
fi

# Build spacesvm, which is run as a subprocess
echo "Building spacesvm in $binary_directory/$name"
go build -o "$binary_directory/$name" ./cmd/spacesvm

mkdir -p ./build
go build -o ./build/spaces-cli ./cmd/spacescli
