# Key-value virtual machine (KVVM)

Avalanche is a network composed of multiple blockchains. Each blockchain is an instance of a [Virtual Machine (VM)](https://docs.avax.network/learn/platform-overview#virtual-machines), much like an object in an object-oriented language is an instance of a class. That is, the VM defines the behavior of the blockchain.

EVM on Avalanche is a canonical use case of virtual machine (VM) that enables smart contracts for decentralized finance applications. KVVM extends beyond smart contracts platform, to provide the key-value storage engine for an ever-growing number of diverse applications, powered by Avalanche protocol.

KVVM defines a blockchain that is a key-value storage server. Each block in the blockchain contains a set of key-value pairs. This VM demonstrates capabilities of custom VMs and custom blockchains. For more information, see: [Create a Virtual Machine](https://docs.avax.network/build/tutorials/platform/create-a-virtual-machine-vm)

KVVM is served over RPC with [go-plugin](https://github.com/hashicorp/go-plugin).

# Quick start

At its core, the Avalanche protocol still maintains the immutable ordered sequence of states in a fully permissionless settings. And KVVM defines the rules and data structures to store key-value pairs.

## Build the `quarkvm` plugin for AvalancheGo

```bash
cd ${HOME}/go/src/github.com/ava-labs/quarkvm
./scripts/build.sh

> Building quarkvm in /Users/patrickogrady/go/src/github.com/ava-labs/avalanchego/build/plugins/tGas3T58KzdjLHhBDMnH2TvrddhqTji5iZAMZ3RXs2NLpSnhH
```

## Generate a Gensis File

```bash
./build/quark-cli genesis

> created genesis and saved to /Users/patrickogrady/code/quarkvm/genesis.json
```

## Clone ava-sim (separate folder)

```bash
git clone https://github.com/ava-labs/ava-sim.git
./scripts/build.sh
```

## Start ava-sim

```bash
./scripts/run.sh /Users/patrickogrady/go/src/github.com/ava-labs/avalanchego/build/plugins/tGas3T58KzdjLHhBDMnH2TvrddhqTji5iZAMZ3RXs2NLpSnhH /Users/patrickogrady/code/quarkvm/genesis.json

>>>>>>
Custom VM endpoints now accessible at:
NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg: http://127.0.0.1:9650/ext/bc/Bbx6eyUCSzoQLzBbM9gnLDdA9HeuiobqQS53iEthvQzeVqbwa
NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ: http://127.0.0.1:9652/ext/bc/Bbx6eyUCSzoQLzBbM9gnLDdA9HeuiobqQS53iEthvQzeVqbwa
NodeID-NFBbbJ4qCmNaCzeW7sxErhvWqvEQMnYcN: http://127.0.0.1:9654/ext/bc/Bbx6eyUCSzoQLzBbM9gnLDdA9HeuiobqQS53iEthvQzeVqbwa
NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu: http://127.0.0.1:9656/ext/bc/Bbx6eyUCSzoQLzBbM9gnLDdA9HeuiobqQS53iEthvQzeVqbwa
NodeID-P7oB2McjBGgW2NXXWVYjV8JEDFoW9xDE5: http://127.0.0.1:9658/ext/bc/Bbx6eyUCSzoQLzBbM9gnLDdA9HeuiobqQS53iEthvQzeVqbwa
```

## Claim a Prefix (work done automatically)

```bash
./build/quark-cli \
--private-key-file .quark-cli-pk \
--url http://127.0.0.1:9650 \
--endpoint /ext/bc/Bbx6eyUCSzoQLzBbM9gnLDdA9HeuiobqQS53iEthvQzeVqbwa  \
claim pat

>>>>>
creating requester with URL http://127.0.0.1:9650 and endpoint "/ext/bc/Bbx6eyUCSzoQLzBbM9gnLDdA9HeuiobqQS53iEthvQzeVqbwa"
Submitting tx NpfRjXRGRCXGxfqq6vcvH3GAm3yijJyxYD7QBxQFDS6YvSnXw with BlockID (zgvHpznxkG7xAh2qgsQFVkrioB4ENdKYfum6KWe6rZGiuzdPf): &{0xc00011a0c8 [175 87 123 222 38 232 10 27 198 13 215 107 60 56 102 21 11 12 195 39 191 122 160 156 155 11 183 164 202 22 22 76 231 28 232 58 18 187 198 249 170 168 232 227 43 85 90 54 94 76 49 184 59 9 194 205 222 162 20 67 208 185 115 12] 0}
issued transaction NpfRjXRGRCXGxfqq6vcvH3GAm3yijJyxYD7QBxQFDS6YvSnXw (success true)
polling transaction "NpfRjXRGRCXGxfqq6vcvH3GAm3yijJyxYD7QBxQFDS6YvSnXw"
confirmed transaction "NpfRjXRGRCXGxfqq6vcvH3GAm3yijJyxYD7QBxQFDS6YvSnXw"
prefix pat info &{Owner:0xc00011dd70 LastUpdated:1638591044 Expiry:1638591074 Keys:1}
```
