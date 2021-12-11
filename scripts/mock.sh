#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -e

if ! [[ "$0" =~ scripts/mock.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi


echo "MOCKGEN RUNNING!"

go install -v github.com/golang/mock/mockgen@v1.6.0

mockgen \
-copyright_file=./LICENSE.header \
-source=./chain/vm.go \
-package=chain \
-destination=./chain/vm_mock.go \
-mock_names=VM=MockVM \
-write_package_comment=true

mockgen \
-copyright_file=./LICENSE.header \
-source=./chain/mempool.go \
-package=chain \
-destination=./chain/mempool_mock.go \
-mock_names=Mempool=MockMempool \
-write_package_comment=false

# https://github.com/golang/mock#reflect-mode
pushd ./chain
mockgen \
-copyright_file=../LICENSE.header \
-package=chain \
-destination=./unsigned_tx_mock.go \
-mock_names=UnsignedTransaction=MockUnsignedTransaction \
-write_package_comment=false \
. \
UnsignedTransaction
popd

echo "MOCKGEN SUCCESS!"
