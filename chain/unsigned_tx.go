// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
)

type UnsignedTransaction interface {
	Copy() UnsignedTransaction
	BlockID() ids.ID
	Prefix() []byte
	Magic() uint64

	FeeUnits(*Genesis) uint64  // number of units to mine tx
	LoadUnits(*Genesis) uint64 // units that should impact fee rate

	ExecuteBase(*Genesis) error
	Execute(*Genesis, database.Database, uint64, ids.ID) error
}
