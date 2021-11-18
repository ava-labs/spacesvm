# Timestamp Virtual Machine

Avalanche is a network composed of multiple blockchains. Each blockchain is an instance of a [Virtual Machine (VM)](https://docs.avax.network/learn/platform-overview#virtual-machines), much like an object in an object-oriented language is an instance of a class. That is, the VM defines the behavior of the blockchain.

TimestampVM defines a blockchain that is a timestamp server. Each block in the blockchain contains the timestamp when it was created along with a 32-byte piece of data (payload). Each block’s timestamp is after its parent’s timestamp. This VM demonstrates capabilities of custom VMs and custom blockchains. For more information, see: [Create a Virtual Machine](https://docs.avax.network/build/tutorials/platform/create-a-virtual-machine-vm)

TimestampVM is served over RPC with [go-plugin](https://github.com/hashicorp/go-plugin).
