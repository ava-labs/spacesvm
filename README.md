# Key-value virtual machine (KVVM)

Avalanche is a network composed of multiple blockchains. Each blockchain is an instance of a [Virtual Machine (VM)](https://docs.avax.network/learn/platform-overview#virtual-machines), much like an object in an object-oriented language is an instance of a class. That is, the VM defines the behavior of the blockchain.

EVM on Avalanche is a canonical use case of virtual machine (VM) that enables smart contracts for decentralized finance applications. KVVM extends beyond smart contracts platform, to provide the key-value storage engine for an ever-growing number of diverse applications, powered by Avalanche protocol.

KVVM defines a blockchain that is a key-value storage server. Each block in the blockchain contains a set of key-value pairs. This VM demonstrates capabilities of custom VMs and custom blockchains. For more information, see: [Create a Virtual Machine](https://docs.avax.network/build/tutorials/platform/create-a-virtual-machine-vm)

KVVM is served over RPC with [go-plugin](https://github.com/hashicorp/go-plugin).

# Quick start

At its core, the Avalanche protocol still maintains the immutable ordered sequence of states in a fully permissionless settings. And KVVM defines the rules and data structures to store key-value pairs.

To interact with Avalanche network RPC chain APIs, download and run a [AvalancheGo](https://github.com/ava-labs/avalanchego#installation) node locally, as follows:

```bash
# run 1 avalanchego node in local network
# TODO: test with 3 nodes?
kill -9 $(lsof -t -i:9650)
kill -9 $(lsof -t -i:9651)
cd ${HOME}/go/src/github.com/ava-labs/avalanchego
./build/avalanchego \
--log-level=info \
--network-id=local \
--public-ip=127.0.0.1 \
--http-port=9650 \
--db-type=memdb \
--staking-enabled=false

# make sure the node is up
curl -X POST --data '{
    "jsonrpc":"2.0",
    "id"     :1,
    "method" :"health.health"
}' -H 'content-type:application/json;' 127.0.0.1:9650/ext/health
```

TODO: example commands


