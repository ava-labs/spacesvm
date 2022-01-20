#!/usr/bin/env bash
set -e

# e.g.,
# ./scripts/run.sh 1.7.3
#
# to shut the cluster down
# SHUTDOWN=false ./scripts/run.sh 1.7.3

# to run E2E tests
# E2E=true ./scripts/run.sh 1.7.3
if ! [[ "$0" =~ scripts/run.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

VERSION=$1
if [[ -z "${VERSION}" ]]; then
  echo "Missing version argument!"
  echo "Usage: ${0} [VERSION]" >> /dev/stderr
  exit 255
fi

SHUTDOWN=${SHUTDOWN:-false}
E2E=${E2E:-false}
if [[ ${SHUTDOWN} == true || ${E2E} == true ]]; then
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

echo "building spacesvm"
go build \
-o /tmp/avalanchego-v${VERSION}/plugins/sqja3uK17MJxfC7AN8nGadBw9JK5BcrsNwNynsqP5Gih8M5Bm \
./cmd/spacesvm
find /tmp/avalanchego-v${VERSION}

echo "building spaces-cli"
go build -v -o /tmp/spaces-cli ./cmd/spaces-cli

echo "creating allocations file"
cat <<EOF > /tmp/allocations.json
[
  {"address":"0xF9370fa73846393798C2d23aa2a4aBA7489d9810", "balance":10000000},
  {"address":"0x8Db3219F3f59b504BCF132EfB4B87Bf08c771d83", "balance":10000000},
  {"address":"0x162a5fadfdd769f9a665701348FbeEd12A4FFce7", "balance":10000000},
  {"address":"0x69fd199Aca8250d520F825d22F4ad9db4A58E9D9", "balance":10000000},
  {"address":"0x454474642C32b19E370d9A55c20431d85833cDD6", "balance":10000000},
  {"address":"0xeB4Fc761FAb7501abe8cD04b2d831a45E8913DdF", "balance":10000000},
  {"address":"0xD23cbfA7eA985213aD81223309f588A7E66A246A", "balance":10000000}
]
EOF

echo "creating VM genesis file"
rm -f /tmp/spacesvm.genesis
/tmp/spaces-cli genesis 1 /tmp/allocations.json \
--genesis-file /tmp/spacesvm.genesis \
--airdrop-hash 0xccbf8e430b30d08b5b3342208781c40b373d1b5885c1903828f367230a2568da \
--airdrop-units 10000

echo "building runner"
pushd ./tests/runner
go build -v -o /tmp/runner .
popd

if [[ ${E2E} == true ]]; then
  echo "building e2e.test"
  # to install the ginkgo binary (required for test build and run)
  go install -v github.com/onsi/ginkgo/v2/ginkgo@v2.0.0-rc2
  ACK_GINKGO_RC=true ginkgo build ./tests/e2e
  ./tests/e2e/e2e.test --help
fi

echo "launch local test cluster in the background"
/tmp/runner \
--avalanchego-path=/tmp/avalanchego-v${VERSION}/avalanchego \
--vm-id=sqja3uK17MJxfC7AN8nGadBw9JK5BcrsNwNynsqP5Gih8M5Bm \
--vm-genesis-path=/tmp/spacesvm.genesis \
--output-path=/tmp/avalanchego-v${VERSION}/output.yaml 2> /dev/null &
PID=${!}

sleep 60
echo "wait until local cluster is ready from PID ${PID}"
while [[ ! -s /tmp/avalanchego-v${VERSION}/output.yaml ]]
  do
  echo "waiting for /tmp/avalanchego-v${VERSION}/output.yaml creation"
  sleep 5
  # wait up to 5-minute
  ((c++)) && ((c==60)) && break
done

if [[ -f "/tmp/avalanchego-v${VERSION}/output.yaml" ]]; then
  echo "cluster is ready!"
  cat /tmp/avalanchego-v${VERSION}/output.yaml
else
  echo "cluster is not ready in time... terminating ${PID}"
  kill ${PID}
  exit 255
fi

if [[ ${E2E} == true ]]; then
  echo "running e2e tests against the local cluster with shutdown flag '${_SHUTDOWN_FLAG}'"
  ./tests/e2e/e2e.test \
  --ginkgo.v \
  --cluster-info-path /tmp/avalanchego-v${VERSION}/output.yaml \
  ${_SHUTDOWN_FLAG}

  echo "ALL SUCCESS!"
fi
