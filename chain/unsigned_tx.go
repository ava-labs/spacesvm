// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto"
)

type UnsignedTransaction interface {
	SetBlockID(block ids.ID)
	GetSender() [crypto.SECP256K1RPKLen]byte
	GetBlockID() ids.ID
	Units() uint64 // number of units to mine tx

	ExecuteBase() error
	Execute(database.Database, uint64) error
}
