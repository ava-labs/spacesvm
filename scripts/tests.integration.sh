#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ scripts/tests.integration.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

# to install the ginkgo binary (required for test build and run)
go install -v github.com/onsi/ginkgo/v2/ginkgo@v2.0.0-rc2

# run with 3 embedded VMs
ACK_GINKGO_RC=true ginkgo \
run \
-v \
./tests/integration \
-- \
--vms 3 \
--min-price 1

echo "ALL SUCCESS!"
