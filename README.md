# SpacesVM 

Avalanche is a network composed of multiple blockchains. Each blockchain is an instance of a [Virtual Machine (VM)](https://docs.avax.network/learn/platform-overview#virtual-machines), much like an object in an object-oriented language is an instance of a class. That is, the VM defines the behavior of the blockchain.

EVM on Avalanche is a canonical use case of virtual machine (VM) that enables smart contracts for decentralized finance applications. KVVM extends beyond smart contracts platform, to provide the key-value storage engine for an ever-growing number of diverse applications, powered by Avalanche protocol.

KVVM defines a blockchain that is a key-value storage server. Each block in the blockchain contains a set of key-value pairs. This VM demonstrates capabilities of custom VMs and custom blockchains. For more information, see: [Create a Virtual Machine](https://docs.avax.network/build/tutorials/platform/create-a-virtual-machine-vm)

KVVM is served over RPC with [go-plugin](https://github.com/hashicorp/go-plugin).

# spacesvm
To build the VM, run `VM=true ./scripts/build.sh`.

# spaces-cli
_To build the CLI, run `./scripts/build.sh`. It will be placed in `./build/spaces-cli` and
`$GOBIN/spaces-cli`._

```
SpacesVM CLI

Usage:
  spaces-cli [command]

Available Commands:
  activity     View recent activity on the network
  claim        Claims the given prefix
  completion   generate the autocompletion script for the specified shell
  create       Creates a new key in the default location
  delete       Deletes a key-value pair for the given prefix
  delete-file  Deletes all hashes reachable from root file identifier
  genesis      Creates a new genesis in the default location
  help         Help about any command
  info         Reads space info and all values at space
  lifeline     Extends the life of a given prefix
  move         Transfers a space to another address
  resolve      Reads a value at space/key
  resolve-file Reads a file at space/key and saves it to disk
  set          Writes a key-value pair for the given prefix
  set-file     Writes a file to the given space
  transfer     Transfers units to another address

Flags:
      --endpoint string           RPC Endpoint for VM (default "https://memeshowdown.com")
  -h, --help                      help for spaces-cli
      --private-key-file string   private key file path (default ".spaces-cli-pk")

Use "spaces-cli [command] --help" for more information about a command.
```

# Public Endpoints (`/public`)

## spacesvm.ping
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.ping",
  "params":{},
  "id": 1
}
>>> {"sucess":<bool>}
```

## spacesvm.genesis
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.genesis",
  "params":{},
  "id": 1
}
>>> {"genesis":<genesis file>}
```

## spacesvm.suggestedFee
_Provide your intent and get back a transaction to sign._
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.suggestedFee",
  "params":{
    "input":<chain.Input (tx abstractor)>
  },
  "id": 1
}
>>> {"typedData":<EIP-712 compliant typed data for signing>,
>>> "totalCost":<uint64>}
```

### chain.Input
```
{
  "type":<string>,
  "space":<string>,
  "key":<string>,
  "value":<base64 encoded>,
  "to":<hex encoded>,
  "units":<uint64>
}
```

#### Transaction Types
```
claim    {type,space}
lifeline {type,space,units}
set      {type,space,key,value}
delete   {type,space,key}
move     {type,space,to}
transfer {type,to,units}

```

## spacesvm.issueTx
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.issueTx",
  "params":{
    "typedData":<EIP-712 compliant typed data>,
    "signature":<hex-encoded sig>
  },
  "id": 1
}
>>> {"txId":<ID>}
```

## spacesvm.hasTx
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.hasTx",
  "params":{
    "txId":<transaction ID>
  },
  "id": 1
}
>>> {"accepted":<bool>}
```

## spacesvm.lastAccepted
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.lastAccepted",
  "params":{},
  "id": 1
}
>>> {"height":<uint64>, "blockId":<ID>}
```

## spacesvm.claimed
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.claimed",
  "params":{
    "space":<string>
  },
  "id": 1
}
>>> {"claimed":<bool>}
```

## spacesvm.info
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.info",
  "params":{
    "space":<string>
  },
  "id": 1
}
>>> {"info":<chain.SpaceInfo>, "values":[<chain.KeyValueMeta>]}
```

### chain.SpaceInfo
```
{
  "owner":<hex encoded>,
  "created":<unix>,
  "updated":<unix>,
  "expiry":<unix>,
  "units":<uint64>,
  "rawSpace":<ShortID>
}
```

### chain.KeyValueMeta
```
{
  "key":<string>,
  "valueMeta":{
    "created":<unix>,
    "updated":<unix>,
    "txId":<ID>, // where value was last set
    "size":<uint64>
  }
}
```

## spacesvm.resolve
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.resolve",
  "params":{
    "path":<string | ex:jim/twitter>
  },
  "id": 1
}
>>> {"exists":<bool>, "value":<base64 encoded>, "valueMeta":<chain.ValueMeta>}
```

## spacesvm.balance
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.balance",
  "params":{
    "address":<hex encoded>
  },
  "id": 1
}
>>> {"balance":<uint64>}
```

## spacesvm.recentActivity
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.recentActivity",
  "params":{},
  "id": 1
}
>>> {"activity":[<chain.Activity>,...]}
```

### chain.Activity
```
{
  "timestamp":<unix>,
  "sender":<address>,
  "txId":<ID>,
  "type":<string>,
  "space":<string>,
  "key":<string>,
  "to":<hex encoded>,
  "units":<uint64>
}
```

#### Activity Types
```
claim    {timestamp,sender,txId,type,space}
lifeline {timestamp,sender,txId,type,space,units}
set      {timestamp,sender,txId,type,space,key,value}
delete   {timestamp,sender,txId,type,space,key}
move     {timestamp,sender,txId,type,space,to}
transfer {timestamp,sender,txId,type,to,units}
reward   {timestamp,txId,type,to,units}
```

# Advanced Public Endpoints (`/public`)

## spacesvm.suggestedRawFee
_Can use this to get the current fee rate._
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.suggestedRawFee",
  "params":{},
  "id": 1
}
>>> {"price":<uint64>,"cost":<uint64>}
```

## spacesvm.issueRawTx
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.issueRawTx",
  "params":{
    "tx":<raw tx bytes>
  },
  "id": 1
}
>>> {"txId":<ID>}
```

# Creating Transactions
```
1) spacesvm.claimed {"space":"patrick"} => Yes/No
2) spacesvm.suggestedFee {"input":{"type":"claim", "space":"patrick"}} => {"typedData":<EIP-712 Typed Data>, "cost":<total fee>}
3) sign EIP-712 Typed Data
4) spacesvm.issueTx {"typedData":<from spacesvm.suggestedFee>, "signature":<sig from step 3>} => {"txId":<ID>}
5) [loop] spacesvm.hasTx {"txId":<ID>} => {"accepted":true"}
```

# Uploading Files
```
spaces-cli set-file patrick ~/Downloads/computer.gif -> patrick/6fe5a52f52b34fb1e07ba90bad47811c645176d0d49ef0c7a7b4b22013f676c8
spaces-cli resolve-file patrick/6fe5a52f52b34fb1e07ba90bad47811c645176d0d49ef0c7a7b4b22013f676c8 computer_copy.gif
spaces-cli delete-file patrick/6fe5a52f52b34fb1e07ba90bad47811c645176d0d49ef0c7a7b4b22013f676c8
```

////// REWRITE ///////////////
# Features
TODO: Extend on
* PoW Transactions (no tokens)
* No Nonces (replay protection from blockId + txId)
* Prefixes (address prefixes reserved)
* Hashed Value Keys
* Prefix Expiry (based on weight of all key-values)
* Load Units vs Fee Units
* Lifeline Rewards (why run a node -> don't need to mine)
* Block Value Reuse

# RPC
## /public
* range query

## /private
* set beneficiary

# Quick start

At its core, the Avalanche protocol still maintains the immutable ordered sequence of states in a fully permissionless settings. And KVVM defines the rules and data structures to store key-value pairs.

## Run `spacesvm` with local network

[`scripts/run.sh`](scripts/run.sh) automatically installs [avalanchego](https://github.com/ava-labs/avalanchego) to set up a local networkand creates a `spacesvm` genesis file. To build and run E2E tests, you need to set the variable `E2E` before it: `E2E=true ./scripts/run.sh 1.7.3`

See [`tests/e2e`](tests/e2e) and [`tests/runner`](tests/runner) to see how it's set up and how its client requests are made:

```bash
# to startup a cluster
cd ${HOME}/go/src/github.com/ava-labs/spacesvm
./scripts/run.sh 1.7.3

# to run full e2e tests and shut down cluster afterwards
cd ${HOME}/go/src/github.com/ava-labs/spacesvm
E2E=true ./scripts/run.sh 1.7.3
```

```bash
# inspect cluster endpoints when ready
cat /tmp/avalanchego-v1.7.3/output.yaml
<<COMMENT
endpoint: /ext/bc/2VCAhX6vE3UnXC6s1CBPE6jJ4c4cHWMfPgCptuWS59pQ9vbeLM
logsDir: ...
pid: 12811
uris:
- http://localhost:56239
- http://localhost:56251
- http://localhost:56253
- http://localhost:56255
- http://localhost:56257
COMMENT

# ping the local cluster
curl --location --request POST 'http://localhost:61858/ext/bc/BJfusM2TpHCEfmt5i7qeE1MwVCbw5jU1TcZNz8MYUwG1PGYRL/public' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "spacesvm.ping",
    "params":{},
    "id": 1
}'
<<COMMENT
{"jsonrpc":"2.0","result":{"success":true},"id":1}
COMMENT

# resolve a path
curl --location --request POST 'http://localhost:61858/ext/bc/BJfusM2TpHCEfmt5i7qeE1MwVCbw5jU1TcZNz8MYUwG1PGYRL/public' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "spacesvm.resolve",
    "params":{
      "path": "patrick.avax/twitter"
    },
    "id": 1
}'
<<COMMENT
{"jsonrpc":"2.0","result":{"exists":true, "value":"QF9wYXRyaWNrb2dyYWR5"},"id":1}
COMMENT

# to terminate the cluster
kill 12811
```

# CLI Usage
## Create Genesis
```bash
./build/spaces-cli genesis
```

## Create Private Key
```bash
./build/spaces-cli create
```

## Claim a Prefix
```bash
./build/spaces-cli \
--private-key-file .spaces-cli-pk \
--endpoint http://localhost:61858/ext/bc/BJfusM2TpHCEfmt5i7qeE1MwVCbw5jU1TcZNz8MYUwG1PGYRL  \
claim patrick.avax

mining in progress[ShWhojqb9FYqf2cWTYWauv1QFuT6igUxLjqATntvq3E52kdLh/201]... (elapsed=1.01s, est. remaining=1m54.3s, threads=16)
mining in progress[ShWhojqb9FYqf2cWTYWauv1QFuT6igUxLjqATntvq3E52kdLh/3329621]... (elapsed=3.01s, est. remaining=1m52.3s, threads=16)
mining in progress[ShWhojqb9FYqf2cWTYWauv1QFuT6igUxLjqATntvq3E52kdLh/6640466]... (elapsed=5.01s, est. remaining=1m50.3s, threads=16)
mining in progress[ShWhojqb9FYqf2cWTYWauv1QFuT6igUxLjqATntvq3E52kdLh/9938320]... (elapsed=7.01s, est. remaining=1m48.3s, threads=16)
mining in progress[ShWhojqb9FYqf2cWTYWauv1QFuT6igUxLjqATntvq3E52kdLh/13222800]... (elapsed=9.01s, est. remaining=1m46.3s, threads=16)
mining in progress[ShWhojqb9FYqf2cWTYWauv1QFuT6igUxLjqATntvq3E52kdLh/16473949]... (elapsed=11.01s, est. remaining=1m44.3s, threads=16)
mining in progress[ShWhojqb9FYqf2cWTYWauv1QFuT6igUxLjqATntvq3E52kdLh/19307265]... (elapsed=13.01s, est. remaining=1m42.3s, threads=16)
mining in progress[ShWhojqb9FYqf2cWTYWauv1QFuT6igUxLjqATntvq3E52kdLh/22151737]... (elapsed=15.01s, est. remaining=1m40.3s, threads=16)
mining in progress[ShWhojqb9FYqf2cWTYWauv1QFuT6igUxLjqATntvq3E52kdLh/25580779]... (elapsed=17.01s, est. remaining=1m38.3s, threads=16)
mining in progress[ShWhojqb9FYqf2cWTYWauv1QFuT6igUxLjqATntvq3E52kdLh/28485504]... (elapsed=19.01s, est. remaining=1m36.3s, threads=16)
mining in progress[ShWhojqb9FYqf2cWTYWauv1QFuT6igUxLjqATntvq3E52kdLh/31685345]... (elapsed=21.01s, est. remaining=1m34.3s, threads=16)
mining in progress[ShWhojqb9FYqf2cWTYWauv1QFuT6igUxLjqATntvq3E52kdLh/34616110]... (elapsed=23.01s, est. remaining=1m32.3s, threads=16)
mining in progress[ShWhojqb9FYqf2cWTYWauv1QFuT6igUxLjqATntvq3E52kdLh/37668727]... (elapsed=25.01s, est. remaining=1m30.3s, threads=16)
mining complete[40596778] (difficulty=188, surplus=541200, elapsed=26.97s)
issuing tx 7Y5voKiHGvytF7ddroV7UvgW8LgmxPR7EzZSyJY1MwQJ4yC9x (fee units=6150, load units=50, difficulty=188, blkID=ShWhojqb9FYqf2cWTYWauv1QFuT6igUxLjqATntvq3E52kdLh)
issued transaction 7Y5voKiHGvytF7ddroV7UvgW8LgmxPR7EzZSyJY1MwQJ4yC9x (now polling)
transaction 7Y5voKiHGvytF7ddroV7UvgW8LgmxPR7EzZSyJY1MwQJ4yC9x confirmed
raw prefix M9Jh5DMRXwMwaTHciFLVAMpc9dZKFpuGE: units=1 expiry=2022-02-09 02:17:33 -0800 PST (719h59m58.807801s remaining)
```

## Set Key in Prefix
```bash
./build/spaces-cli \
--private-key-file .spaces-cli-pk \
--endpoint http://localhost:61858/ext/bc/BJfusM2TpHCEfmt5i7qeE1MwVCbw5jU1TcZNz8MYUwG1PGYRL  \
set patrick.avax/twitter @_patrickogrady

mining in progress[2QsEbN4VgFjeMMfxU1T9KWUFMLtyTpYT7Ud8fw4kZ7hZfMMWhA/37]... (elapsed=1.01s, threads=16)
mining complete[1145609] (difficulty=165, surplus=715, elapsed=1.76s)
issuing tx APWDpcgjUcDDP8P4L97x3BbKkvJ4NzZfESXc2AcNjQV99aRqw (fee units=11, load units=11, difficulty=165, blkID=2QsEbN4VgFjeMMfxU1T9KWUFMLtyTpYT7Ud8fw4kZ7hZfMMWhA)
issued transaction APWDpcgjUcDDP8P4L97x3BbKkvJ4NzZfESXc2AcNjQV99aRqw (now polling)
transaction APWDpcgjUcDDP8P4L97x3BbKkvJ4NzZfESXc2AcNjQV99aRqw confirmed
raw prefix M9Jh5DMRXwMwaTHciFLVAMpc9dZKFpuGE: units=2 expiry=2022-01-25 02:18:47 -0800 PST (359h59m58.948798s remaining)
```

## Get Key in Preifx
```bash
./build/spaces-cli \
--private-key-file .spaces-cli-pk \
--endpoint http://localhost:61858/ext/bc/BJfusM2TpHCEfmt5i7qeE1MwVCbw5jU1TcZNz8MYUwG1PGYRL  \
get patrick.avax/twitter

range success 1 key-values
key: "twitter", value: "@_patrickogrady"
```

## Delete Key in Preifx
```bash
./build/spaces-cli \
--private-key-file .spaces-cli-pk \
--endpoint http://localhost:61858/ext/bc/BJfusM2TpHCEfmt5i7qeE1MwVCbw5jU1TcZNz8MYUwG1PGYRL  \
delete patrick.avax/twitter

mining in progress[g5rmmSRrCMZDUsg5KBzNR4wpeX6Ph6xaXoNCPwUWj7xHM7epD/333]... (elapsed=1.01s, threads=16)
mining complete[222163] (difficulty=127, surplus=297, elapsed=1.16s)
issuing tx 2AmF6zY1mTdniwXrifoKCfzPGEqrKyJv21S8k5gSa1MYbhFR3h (fee units=11, load units=11, difficulty=127, blkID=g5rmmSRrCMZDUsg5KBzNR4wpeX6Ph6xaXoNCPwUWj7xHM7epD)
issued transaction 2AmF6zY1mTdniwXrifoKCfzPGEqrKyJv21S8k5gSa1MYbhFR3h (now polling)
transaction 2AmF6zY1mTdniwXrifoKCfzPGEqrKyJv21S8k5gSa1MYbhFR3h confirmed
raw prefix M9Jh5DMRXwMwaTHciFLVAMpc9dZKFpuGE: units=1 expiry=2022-02-09 02:20:55 -0800 PST (719h59m58.687729s remaining)
```

## Extend Prefix Life
```bash
./build/spaces-cli \
--private-key-file .spaces-cli-pk \
--endpoint http://localhost:61858/ext/bc/BJfusM2TpHCEfmt5i7qeE1MwVCbw5jU1TcZNz8MYUwG1PGYRL  \
lifeline patrick.avax

mining in progress[2ty1GmQAatedGd3CeUXzj5YUYVaqeMawpvCRPxqCf12u7fNfM/39]... (elapsed=1.01s, threads=16)
mining complete[469870] (difficulty=169, surplus=690, elapsed=1.31s)
issuing tx 2bJPjSWyr6NoDUE9ZyyamDetjNYDQ87G4dtZnT3LVf4uGYtFtU (fee units=10, load units=10, difficulty=169, blkID=2ty1GmQAatedGd3CeUXzj5YUYVaqeMawpvCRPxqCf12u7fNfM)
issued transaction 2bJPjSWyr6NoDUE9ZyyamDetjNYDQ87G4dtZnT3LVf4uGYtFtU (now polling)
transaction 2bJPjSWyr6NoDUE9ZyyamDetjNYDQ87G4dtZnT3LVf4uGYtFtU confirmed
raw prefix M9Jh5DMRXwMwaTHciFLVAMpc9dZKFpuGE: units=1 expiry=2022-02-09 04:07:07 -0800 PST (721h44m47.312056s remaining)
```

# HTTP Examples
## Get Prefix Info
_cGF0cmljay5hdmF4 is "patrick.avax" in base64_
```bash
curl --location --request POST 'http://localhost:61858/ext/bc/BJfusM2TpHCEfmt5i7qeE1MwVCbw5jU1TcZNz8MYUwG1PGYRL/public' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "spacesvm.prefixInfo",
    "params":{
        "prefix":"cGF0cmljay5hdmF4"
    },
    "id": 1
}'

{
    "jsonrpc": "2.0",
    "result": {
        "info": {
            "owner": [
                3,
                74,
                255,
                247,
                51,
                219,
                231,
                3,
                243,
                231,
                100,
                99,
                245,
                34,
                43,
                222,
                16,
                61,
                202,
                99,
                39,
                113,
                85,
                197,
                4,
                185,
                122,
                214,
                117,
                141,
                45,
                98,
                196
            ],
            "created": 1641809853,
            "lastUpdated": 1641810055,
            "expiry": 1644408427,
            "units": 1,
            "rawPrefix": "M9Jh5DMRXwMwaTHciFLVAMpc9dZKFpuGE"
        }
    },
    "id": 1
}
```

# Difficulty Estiamtes
To see what performance you can get, run:
```bash
go test -bench=. ./pow/...
```

Here are some example results:
```bash
goos: darwin
goarch: amd64
pkg: github.com/ava-labs/spacesvm/pow
cpu: Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz
BenchmarkDifficulty1-16       	    1300	    921956 ns/op
BenchmarkDifficulty10-16      	     100	  11083185 ns/op
BenchmarkDifficulty50-16      	     100	  50243796 ns/op
BenchmarkDifficulty100-16     	      12	  84529354 ns/op
BenchmarkDifficulty500-16     	       7	 251526615 ns/op
BenchmarkDifficulty1000-16    	       7	 571905766 ns/op
```
