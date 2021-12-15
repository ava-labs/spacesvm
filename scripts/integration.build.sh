#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ scripts/integration.build.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

# go get -v github.com/onsi/ginkgo/v2@v2.0.0-rc2
go install -v github.com/onsi/ginkgo/v2/ginkgo@v2.0.0-rc2

ACK_GINKGO_RC=true ginkgo build ./tests/integration
./tests/integration/integration.test --help
