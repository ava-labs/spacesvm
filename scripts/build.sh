#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if ! [[ "$0" =~ scripts/build.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

# Set default binary directory location
name="sqja3uK17MJxfC7AN8nGadBw9JK5BcrsNwNynsqP5Gih8M5Bm"

# Build spacesvm, which is run as a subprocess
mkdir -p ./build

echo "Building spacesvm in ./build/$name"
go build -o ./build/$name ./cmd/spacesvm

echo "Building spaces-cli in ./build/spaces-cli"
go build -o ./build/spaces-cli ./cmd/spaces-cli
