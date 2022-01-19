# SpacesVM

Avalanche is a network composed of multiple blockchains. Each blockchain is an instance
of a [Virtual Machine (VM)](https://docs.avax.network/learn/platform-overview#virtual-machines),
much like an object in an object-oriented language is an instance of a class. That is,
the VM defines the behavior of the blockchain where it is instantiated. The use of
[Coreth (EVM)](https://github.com/ava-labs/coreth) on the [Avalanche C-Chain](https://docs.avax.network/learn/platform-overview)
is a canonical use case of a virtual machine (EVM) and its instantiation (C-Chain) on the Primary Subnet (Avalanche Mainnet). One
could deploy their own instance of the EVM as their own blockchain (to take
this to its logical conclusion.

Just as Coreth powers the C-Chain, SpacesVM can be used to power its own
blockchain. However, instead of providing a place to execute smart contracts on
decentralized applications, SpacesVM enables anyone to store arbitrary data for
fast retrieval, like a Key-Value Database where a single party controls an
entire hierarchy of keys, you can claim your own hierarchy. (TODO).

You could build...
* Name Service
* Link Service
* dApp Metadata Backend
* Twitter Feed-like
* NFT Storage (value hashing)

## How it Works
### Action Types
#### Claim

##### Community support

#### Set/Delete

##### Arbitrary Size File Support (using CLI)

#### Resolve

#### Transfer

#### Move

### Wallet Support: `eth_typedSignedData`
TODO: Insert image of signing using MM

### Fee Mechanisms
Claim Desirability + Decay Rate
FeeUnits vs Load Units vs Expiry Units (per action)
Expiry Rate vs Units

### Space Rewards
Lottery allocation X% of fee

### Genesis Allocation
Airdrop `10,000 SPC` for anyone who has interacted with C-Chain more than
twice.

## Usage
_If you are interested in running the VM, not using it. Jump to [Running the
VM](#running-the-vm)._

Public Beta...

### tryspaces.xyz
What better way to understand how this works than to see it in action?

TODO: insert try spaces image + link

Hooked up to public beta

### spaces-cli
_To build the CLI, run `./scripts/build.sh`. It will be placed in `./build/spaces-cli` and
`$GOBIN/spaces-cli`._

```
SpacesVM CLI

Usage:
  spaces-cli [command]

Available Commands:
  activity     View recent activity on the network
  claim        Claims the given space
  completion   generate the autocompletion script for the specified shell
  create       Creates a new key in the default location
  delete       Deletes a key-value pair for the given space
  delete-file  Deletes all hashes reachable from root file identifier
  genesis      Creates a new genesis in the default location
  help         Help about any command
  info         Reads space info and all values at space
  lifeline     Extends the life of a given space
  move         Transfers a space to another address
  network      View information about this instance of the SpacesVM
  owned        Fetches all owned spaces for the address associated with the private key
  resolve      Reads a value at space/key
  resolve-file Reads a file at space/key and saves it to disk
  set          Writes a key-value pair for the given space
  set-file     Writes a file to the given space
  transfer     Transfers units to another address

Flags:
      --endpoint string           RPC endpoint for VM (default "https://api.tryspaces.xyz")
  -h, --help                      help for spaces-cli
      --private-key-file string   private key file path (default ".spaces-cli-pk")
      --verbose                   Print verbose information about operations

Use "spaces-cli [command] --help" for more information about a command.
```

#### Uploading Files
```
spaces-cli set-file patrick ~/Downloads/computer.gif -> patrick/6fe5a52f52b34fb1e07ba90bad47811c645176d0d49ef0c7a7b4b22013f676c8
spaces-cli resolve-file patrick/6fe5a52f52b34fb1e07ba90bad47811c645176d0d49ef0c7a7b4b22013f676c8 computer_copy.gif
spaces-cli delete-file patrick/6fe5a52f52b34fb1e07ba90bad47811c645176d0d49ef0c7a7b4b22013f676c8
```

### [Golang SDK](https://github.com/ava-labs/spacesvm/blob/master/client/client.go)
```golang
// Client defines spacesvm client operations.
type Client interface {
	// Pings the VM.
	Ping() (bool, error)
	// Network information about this instance of the VM
	Network() (uint32, ids.ID, ids.ID, error)

	// Returns the VM genesis.
	Genesis() (*chain.Genesis, error)
	// Accepted fetches the ID of the last accepted block.
	Accepted() (ids.ID, error)

	// Returns if a space is already claimed
	Claimed(space string) (bool, error)
	// Returns the corresponding space information.
	Info(space string) (*chain.SpaceInfo, []*chain.KeyValueMeta, error)
	// Balance returns the balance of an account
	Balance(addr common.Address) (bal uint64, err error)
	// Resolve returns the value associated with a path
	Resolve(path string) (exists bool, value []byte, valueMeta *chain.ValueMeta, err error)

	// Requests the suggested price and cost from VM.
	SuggestedRawFee() (uint64, uint64, error)
	// Issues the transaction and returns the transaction ID.
	IssueRawTx(d []byte) (ids.ID, error)

	// Requests the suggested price and cost from VM, returns the input as
	// TypedData.
	SuggestedFee(i *chain.Input) (*tdata.TypedData, uint64, error)
	// Issues a human-readable transaction and returns the transaction ID.
	IssueTx(td *tdata.TypedData, sig []byte) (ids.ID, error)

	// Checks the status of the transaction, and returns "true" if confirmed.
	HasTx(id ids.ID) (bool, error)
	// Polls the transactions until its status is confirmed.
	PollTx(ctx context.Context, txID ids.ID) (confirmed bool, err error)

	// Recent actions on the network (sorted from recent to oldest)
	RecentActivity() ([]*chain.Activity, error)
}
```

### Public Endpoints (`/public`)

#### spacesvm.ping
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.ping",
  "params":{},
  "id": 1
}
>>> {"success":<bool>}
```

#### spacesvm.network
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.network",
  "params":{},
  "id": 1
}
>>> {"networkId":<uint32>, "subnetId":<ID>, "chainId":<ID>}
```

#### spacesvm.genesis
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

#### spacesvm.suggestedFee
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

##### chain.Input
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

###### Transaction Types
```
claim    {type,space}
lifeline {type,space,units}
set      {type,space,key,value}
delete   {type,space,key}
move     {type,space,to}
transfer {type,to,units}

```

#### spacesvm.issueTx
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

##### Transaction Creation Worflow
```
1) spacesvm.claimed {"space":"patrick"} => Yes/No
2) spacesvm.suggestedFee {"input":{"type":"claim", "space":"patrick"}} => {"typedData":<EIP-712 Typed Data>, "cost":<total fee>}
3) sign EIP-712 Typed Data
4) spacesvm.issueTx {"typedData":<from spacesvm.suggestedFee>, "signature":<sig from step 3>} => {"txId":<ID>}
5) [loop] spacesvm.hasTx {"txId":<ID>} => {"accepted":true"}
```

#### spacesvm.hasTx
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

#### spacesvm.lastAccepted
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

#### spacesvm.claimed
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

#### spacesvm.info
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

##### chain.SpaceInfo
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

##### chain.KeyValueMeta
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

#### spacesvm.resolve
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

#### spacesvm.balance
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

#### spacesvm.recentActivity
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

##### chain.Activity
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

###### Activity Types
```
claim    {timestamp,sender,txId,type,space}
lifeline {timestamp,sender,txId,type,space,units}
set      {timestamp,sender,txId,type,space,key,value}
delete   {timestamp,sender,txId,type,space,key}
move     {timestamp,sender,txId,type,space,to}
transfer {timestamp,sender,txId,type,to,units}
reward   {timestamp,txId,type,to,units}
```

#### spacesvm.owned
```
<<< POST
{
  "jsonrpc": "2.0",
  "method": "spacesvm.owned",
  "params":{
    "address":<hex encoded>
  },
  "id": 1
}
>>> {"spaces":[<string>]}
```

### Advanced Public Endpoints (`/public`)

#### spacesvm.suggestedRawFee
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

#### spacesvm.issueRawTx
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

## Running the VM
To build the VM, run `VM=true ./scripts/build.sh`.

### Joining the public beta
Put spacesvm binary in plugins dir
Add subnet-id to whitelisted-subnets

TODO: set bootstrap nodes

Here is an example config file:
--network-id=fuji

Make sure to add these commands when running the node:
--config-file

If you'd like to become a validator, reach out to @\_patrickogrady on Twitter
after you've joined the network and synced to tip. Please send a screenshot
indicating you've done this successfully.

### Running a local network
[`scripts/run.sh`](scripts/run.sh) automatically installs [avalanchego](https://github.com/ava-labs/avalanchego) to set up a local network
and creates a `spacesvm` genesis file. To build and run E2E tests, you need to set the variable `E2E` before it: `E2E=true ./scripts/run.sh 1.7.4`

See [`tests/e2e`](tests/e2e) and [`tests/runner`](tests/runner) to see how it's set up and how its client requests are made:

```bash
# to startup a cluster
cd ${HOME}/go/src/github.com/ava-labs/spacesvm
./scripts/run.sh 1.7.4

# to run full e2e tests and shut down cluster afterwards
cd ${HOME}/go/src/github.com/ava-labs/spacesvm
E2E=true ./scripts/run.sh 1.7.4
```

```bash
# inspect cluster endpoints when ready
cat /tmp/avalanchego-v1.7.4/output.yaml
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
      "path": "coolperson/twitter"
    },
    "id": 1
}'
<<COMMENT
{"jsonrpc":"2.0","result":{"exists":true, "value":"....", "valueMeta":{....}},"id":1}
COMMENT

# to terminate the cluster
kill 12811
```
