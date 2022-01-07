#!/usr/bin/env bash
set -e

# e.g.,
# ./scripts/tests.e2e.sh 1.7.3
#
# to keep the cluster alive
# SHUTDOWN=false ./scripts/tests.e2e.sh 1.7.3
if ! [[ "$0" =~ scripts/tests.e2e.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

VERSION=$1
if [[ -z "${VERSION}" ]]; then
  echo "Missing version argument!"
  echo "Usage: ${0} [VERSION]" >> /dev/stderr
  exit 255
fi

SHUTDOWN=${SHUTDOWN:-true}
if [[ ${SHUTDOWN} == true ]]; then
  _SHUTDOWN_FLAG="--shutdown"
else
  _SHUTDOWN_FLAG=""
fi

# download avalanchego
# https://github.com/ava-labs/avalanchego/releases
GOARCH=$(go env GOARCH)
GOOS=$(go env GOOS)
DOWNLOAD_URL=https://github.com/ava-labs/avalanchego/releases/download/v${VERSION}/avalanchego-linux-${GOARCH}-v${VERSION}.tar.gz
DOWNLOAD_PATH=/tmp/avalanchego.tar.gz
if [[ ${GOOS} == "darwin" ]]; then
  DOWNLOAD_URL=https://github.com/ava-labs/avalanchego/releases/download/v${VERSION}/avalanchego-macos-v${VERSION}.zip
  DOWNLOAD_PATH=/tmp/avalanchego.zip
fi

rm -rf /tmp/avalanchego-v${VERSION}
rm -rf /tmp/avalanchego-build
rm -f ${DOWNLOAD_PATH}

echo "downloading avalanchego ${VERSION} at ${DOWNLOAD_URL}"
curl -L ${DOWNLOAD_URL} -o ${DOWNLOAD_PATH}

echo "extracting downloaded avalanchego"
if [[ ${GOOS} == "linux" ]]; then
  tar xzvf ${DOWNLOAD_PATH} -C /tmp
elif [[ ${GOOS} == "darwin" ]]; then
  unzip ${DOWNLOAD_PATH} -d /tmp/avalanchego-build
  mv /tmp/avalanchego-build/build /tmp/avalanchego-v${VERSION}
fi
find /tmp/avalanchego-v${VERSION}

echo "building quarkvm"
go build \
-o /tmp/avalanchego-v${VERSION}/plugins/tGas3T58KzdjLHhBDMnH2TvrddhqTji5iZAMZ3RXs2NLpSnhH \
./cmd/quarkvm
find /tmp/avalanchego-v${VERSION}

echo "building quark-cli"
go build -v -o /tmp/quark-cli ./cmd/quarkcli

echo "creating VM genesis file"
rm -f /tmp/quarkvm.genesis
/tmp/quark-cli genesis --genesis-file /tmp/quarkvm.genesis

echo "building runner"
pushd ./tests/runner
go build -v -o /tmp/runner .
popd

echo "building e2e.test"
# to install the ginkgo binary (required for test build and run)
go install -v github.com/onsi/ginkgo/v2/ginkgo@v2.0.0-rc2
ACK_GINKGO_RC=true ginkgo build ./tests/e2e
./tests/e2e/e2e.test --help

echo "launch local test cluster in the background"
/tmp/runner \
--avalanchego-path=/tmp/avalanchego-v${VERSION}/avalanchego \
--vm-id=tGas3T58KzdjLHhBDMnH2TvrddhqTji5iZAMZ3RXs2NLpSnhH \
--vm-genesis-path=/tmp/quarkvm.genesis \
--output-path=/tmp/avalanchego-v${VERSION}/output.yaml 2> /dev/null &
PID=${!}

sleep 60
echo "wait until local cluster is ready from PID ${PID}"
while [[ ! -s /tmp/avalanchego-v${VERSION}/output.yaml ]]
  do
  echo "waiting for /tmp/avalanchego-v${VERSION}/output.yaml creation"
  sleep 3
done
echo "found it!"
cat /tmp/avalanchego-v${VERSION}/output.yaml

echo "running e2e tests against the local cluster with shutdown flag '${_SHUTDOWN_FLAG}'"
./tests/e2e/e2e.test \
--ginkgo.v \
--cluster-info-path /tmp/avalanchego-v${VERSION}/output.yaml \
${_SHUTDOWN_FLAG}

echo "ALL SUCCESS!"
