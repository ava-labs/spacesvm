// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/quarkvm/crypto"
)

type UnsignedTransaction interface {
	SetBlockID(block ids.ID)
	SetGraffiti(graffiti uint64)
	GetSender() [crypto.PublicKeySize]byte
	GetBlockID() ids.ID

	ExecuteBase() error
	Execute(database.Database, int64) error
}
