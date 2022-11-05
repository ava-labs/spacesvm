#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ scripts/tests.unit.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

SPACESVM_PATH=$(
  cd "$(dirname "${BASH_SOURCE[0]}")"
  cd .. && pwd
)
source "$SPACESVM_PATH"/scripts/constants.sh

go test -race -timeout="3m" -coverprofile="coverage.out" -covermode="atomic" $(go list ./... | grep -v tests)
