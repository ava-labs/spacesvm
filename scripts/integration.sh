#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ scripts/integration.sh ]]; then
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
--min-difficulty 1 \
--min-block-cost 0

# to run with external endpoints
# ACK_GINKGO_RC=true ginkgo run -v --focus "\[Ping\]" ./tests/integration -- \
# --uris http://127.0.0.1:9650 \
# --endpoint /ext/bc/293WF7H8ZEWQcBSTHASC3DWiKGYNTT3siGP52LkgcioeY8nazY
#
# or
#
# ./tests/integration/integration.test --ginkgo.v --ginkgo.focus "\[Ping\]" \
# --uris http://127.0.0.1:9650 \
# --endpoint /ext/bc/293WF7H8ZEWQcBSTHASC3DWiKGYNTT3siGP52LkgcioeY8nazY
