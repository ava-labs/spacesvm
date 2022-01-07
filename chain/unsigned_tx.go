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
	SetGraffiti(graffiti uint64)
	GetSender() [crypto.SECP256K1RPKLen]byte
	GetBlockID() ids.ID
	// Returns the expiry and "true" if applicable.
	// Otherwise returns false.
	GetExpiry() (uint64, bool)

	ExecuteBase() error
	Execute(database.Database, int64) error
}
