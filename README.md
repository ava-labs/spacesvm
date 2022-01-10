# Key-value virtual machine (KVVM)

Avalanche is a network composed of multiple blockchains. Each blockchain is an instance of a [Virtual Machine (VM)](https://docs.avax.network/learn/platform-overview#virtual-machines), much like an object in an object-oriented language is an instance of a class. That is, the VM defines the behavior of the blockchain.

EVM on Avalanche is a canonical use case of virtual machine (VM) that enables smart contracts for decentralized finance applications. KVVM extends beyond smart contracts platform, to provide the key-value storage engine for an ever-growing number of diverse applications, powered by Avalanche protocol.

KVVM defines a blockchain that is a key-value storage server. Each block in the blockchain contains a set of key-value pairs. This VM demonstrates capabilities of custom VMs and custom blockchains. For more information, see: [Create a Virtual Machine](https://docs.avax.network/build/tutorials/platform/create-a-virtual-machine-vm)

KVVM is served over RPC with [go-plugin](https://github.com/hashicorp/go-plugin).

# Features
TODO: Extend on
* PoW Transactions (no tokens)
* Prefixes (address prefixes reserved)
* Hashed Value Keys
* Prefix Expiry (based on weight of all key-values)
* Load Units vs Fee Units
* Lifeline Rewards (why run a node -> don't need to mine)

# RPC
## /public
* range query

## /private
* set beneficiary

# Quick start

At its core, the Avalanche protocol still maintains the immutable ordered sequence of states in a fully permissionless settings. And KVVM defines the rules and data structures to store key-value pairs.

## Run `quarkvm` with local network

[`scripts/tests.e2e.sh`](scripts/tests.e2e.sh) automatically installs [avalanchego](https://github.com/ava-labs/avalanchego) to set up a local network, creates `quarkvm` genesis file, and run e2e tests.

See [`tests/e2e`](tests/e2e) and [`tests/runner`](tests/runner) to see how it's set up and how its client requests are made:

```bash
# to run full e2e tests and shut down cluster afterwards
cd ${HOME}/go/src/github.com/ava-labs/quarkvm
./scripts/tests.e2e.sh 1.7.3

# to run full e2e tests and keep the cluster alive
cd ${HOME}/go/src/github.com/ava-labs/quarkvm
SHUTDOWN=false ./scripts/tests.e2e.sh 1.7.3
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
curl -X POST --data '{
    "jsonrpc": "2.0",
    "method": "quarkvm.ping",
    "params":{},
    "id": 1
}' -H 'content-type:application/json;' 127.0.0.1:56239/ext/bc/2VCAhX6vE3UnXC6s1CBPE6jJ4c4cHWMfPgCptuWS59pQ9vbeLM
<<COMMENT
{"jsonrpc":"2.0","result":{"success":true},"id":1}
COMMENT

# to terminate the cluster
kill 12811
```

## Claim a Prefix (work done automatically)

```bash
./build/quark-cli \
--private-key-file .quark-cli-pk \
--url http://127.0.0.1:9650 \
--endpoint /ext/bc/2VCAhX6vE3UnXC6s1CBPE6jJ4c4cHWMfPgCptuWS59pQ9vbeLM  \
claim pat

>>>>>
creating requester with URL http://127.0.0.1:9650 and endpoint "/ext/bc/2VCAhX6vE3UnXC6s1CBPE6jJ4c4cHWMfPgCptuWS59pQ9vbeLM"
Submitting tx NpfRjXRGRCXGxfqq6vcvH3GAm3yijJyxYD7QBxQFDS6YvSnXw with BlockID (zgvHpznxkG7xAh2qgsQFVkrioB4ENdKYfum6KWe6rZGiuzdPf): &{0xc00011a0c8 [175 87 123 222 38 232 10 27 198 13 215 107 60 56 102 21 11 12 195 39 191 122 160 156 155 11 183 164 202 22 22 76 231 28 232 58 18 187 198 249 170 168 232 227 43 85 90 54 94 76 49 184 59 9 194 205 222 162 20 67 208 185 115 12] 0}
issued transaction NpfRjXRGRCXGxfqq6vcvH3GAm3yijJyxYD7QBxQFDS6YvSnXw (success true)
polling transaction "NpfRjXRGRCXGxfqq6vcvH3GAm3yijJyxYD7QBxQFDS6YvSnXw"
confirmed transaction "NpfRjXRGRCXGxfqq6vcvH3GAm3yijJyxYD7QBxQFDS6YvSnXw"
prefix pat info &{Owner:0xc00011dd70 LastUpdated:1638591044 Expiry:1638591074 Keys:1}
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
pkg: github.com/ava-labs/quarkvm/pow
cpu: Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz
BenchmarkDifficulty1-16       	 1145233	      1047 ns/op
BenchmarkDifficulty10-16      	  113216	     10335 ns/op
BenchmarkDifficulty50-16      	   23011	     52336 ns/op
BenchmarkDifficulty100-16     	   11540	    102029 ns/op
BenchmarkDifficulty500-16     	    1962	    535265 ns/op
BenchmarkDifficulty1000-16    	    1132	   1082758 ns/op
```
