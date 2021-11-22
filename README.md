# Key-value virtual machine (KVVM)

Avalanche is a network composed of multiple blockchains. Each blockchain is an instance of a [Virtual Machine (VM)](https://docs.avax.network/learn/platform-overview#virtual-machines), much like an object in an object-oriented language is an instance of a class. That is, the VM defines the behavior of the blockchain.

EVM on Avalanche is a canonical use case of virtual machine (VM) that enables smart contracts for decentralized finance applications. KVVM extends beyond smart contracts platform, to provide the key-value storage engine for an ever-growing number of diverse applications, powered by Avalanche protocol.

KVVM defines a blockchain that is a key-value storage server. Each block in the blockchain contains a set of key-value pairs. This VM demonstrates capabilities of custom VMs and custom blockchains. For more information, see: [Create a Virtual Machine](https://docs.avax.network/build/tutorials/platform/create-a-virtual-machine-vm)

KVVM is served over RPC with [go-plugin](https://github.com/hashicorp/go-plugin).

# Quick start

At its core, the Avalanche protocol still maintains the immutable ordered sequence of states in a fully permissionless settings. And KVVM defines the rules and data structures to store key-value pairs.

Build quarkvm:

```bash
cd ${HOME}/go/src/github.com/ava-labs/quarkvm
./scripts/build.sh
```
## TODO: MIGRATE TO USING AVA-SIM
*Step 1.* To interact with Avalanche network RPC chain APIs, download and run a [AvalancheGo](https://github.com/ava-labs/avalanchego#installation) node locally, as follows:

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

*Step 2.* Create a user:

```bash
curl --location --request POST '127.0.0.1:9650/ext/keystore' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc":"2.0",
    "id"     :1,
    "method" :"keystore.createUser",
    "params" :{
        "username":"testusername123",
        "password":"insecurestring789"
    }
}'
```

*Step 3.* Import the pre-funded key for the P-chain:

```bash
curl --location --request POST '127.0.0.1:9650/ext/P' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc":"2.0",
    "id"     :1,
    "method" :"platform.importKey",
    "params" :{
        "username":"testusername123",
        "password":"insecurestring789",
        "privateKey":"PrivateKey-ewoqjP7PxY4yr3iLTpLisriqt94hdyDFNgchSxGGztUrTXtNN"
    }
}'
# {"jsonrpc":"2.0","result":{"address":"P-local18jma8ppw3nhx5r4ap8clazz0dps7rv5u00z96u"},"id":1}
```

*Step 4.* Get the list of P-chain addresses:

```bash
curl -X POST --data '{
    "jsonrpc": "2.0",
    "method": "platform.listAddresses",
    "params": {
        "username":"testusername123",
        "password":"insecurestring789"
    },
    "id": 1
}' -H 'content-type:application/json;' 127.0.0.1:9650/ext/P
# {"jsonrpc":"2.0","result":{"addresses":["P-local18jma8ppw3nhx5r4ap8clazz0dps7rv5u00z96u"]},"id":1}
```

*Step 5.* Create a subnet:

```bash
curl --location --request POST '127.0.0.1:9650/ext/P' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc":"2.0",
    "id"     :1,
    "method" :"platform.createSubnet",
    "params" :{
        "username":"testusername123",
        "password":"insecurestring789",
        "threshold":1,
        "controlKeys":["P-local18jma8ppw3nhx5r4ap8clazz0dps7rv5u00z96u"]
    }
}'
# {"jsonrpc":"2.0","result":{"txID":"29uVeLPJB1eQJkzRemU8g8wZDw5uJRqpab5U2mX9euieVwiEbL","changeAddr":"P-local18jma8ppw3nhx5r4ap8clazz0dps7rv5u00z96u"},"id":1}
# 29uVeLPJB1eQJkzRemU8g8wZDw5uJRqpab5U2mX9euieVwiEbL is the subnet blockchain ID
```

*Step 6.* Create a blockchain:

```bash
# TODO: where to get vmID
curl --location --request POST '127.0.0.1:9650/ext/P' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc":"2.0",
    "id"     :1,
    "method" :"platform.createBlockchain",
    "params" :{
        "username":"testusername123",
        "password":"insecurestring789",
        "vmID":"tGas3T58KzdjLHhBDMnH2TvrddhqTji5iZAMZ3RXs2NLpSnhH",
        "subnetID":"29uVeLPJB1eQJkzRemU8g8wZDw5uJRqpab5U2mX9euieVwiEbL",
        "name":"quarkvm",
        "genesisData":"",
        "controlKeys":["P-local18jma8ppw3nhx5r4ap8clazz0dps7rv5u00z96u"]
    }
}'
#
```

*Step 7.* Interact with quarkVM using quark-cli:

```bash
# TODO: add config file/env for key location and/or RPC
quark-cli create
quark-cli claim jim.avax
quark-cli set jim.avax/twitter @jimbo
quark-cli lifeline jim.avax
quark-cli get jim.avax/twitter
quark-cli info jim.avax (remaining life, num keys, claimed/unclaimed/expired)
quark-cli keys jim.avax (get all keys values)
```
