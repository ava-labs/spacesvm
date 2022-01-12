// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto"
)

type UnsignedTransaction interface {
	Copy() UnsignedTransaction
	SetBlockID(block ids.ID)
	SetGraffiti(graffiti uint64)
	GetSender() [crypto.SECP256K1RPKLen]byte
	GetBlockID() ids.ID

	FeeUnits(*Genesis) uint64  // number of units to mine tx
	LoadUnits(*Genesis) uint64 // units that should impact fee rate

	ExecuteBase() error
	Execute(*Genesis, database.Database, uint64, ids.ID) error
}
