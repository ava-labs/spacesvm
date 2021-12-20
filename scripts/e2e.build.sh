#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ scripts/e2e.build.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

# to add dependency (not needed for test build)
# go get -v github.com/onsi/ginkgo/v2@v2.0.0-rc2

# to install the ginkgo binary (required for test build and run)
go install -v github.com/onsi/ginkgo/v2/ginkgo@v2.0.0-rc2

ACK_GINKGO_RC=true ginkgo build ./tests/e2e
./tests/e2e/e2e.test --help
