// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto"

	"github.com/ava-labs/quarkvm/parser"
)

var emptyPublicKeyBytes [crypto.SECP256K1RPKLen]byte

type BaseTx struct {
	Sender   [crypto.SECP256K1RPKLen]byte `serialize:"true" json:"sender"`
	Graffiti uint64                       `serialize:"true" json:"graffiti"`
	BlockID  ids.ID                       `serialize:"true" json:"blockId"`

	// Prefix is the namespace for the "PrefixInfo"
	// whose owner can write and read value for the
	// specific key space.
	// The prefix must not have the delimiter '/' as suffix.
	// Otherwise, the verification will fail.
	Prefix []byte `serialize:"true" json:"prefix"`
}

func (b *BaseTx) SetBlockID(blockID ids.ID) {
	b.BlockID = blockID
}

func (b *BaseTx) SetGraffiti(graffiti uint64) {
	b.Graffiti = graffiti
}

func (b *BaseTx) GetBlockID() ids.ID {
	return b.BlockID
}

func (b *BaseTx) GetSender() [crypto.SECP256K1RPKLen]byte {
	return b.Sender
}

func (b *BaseTx) GetExpiry() (uint64, bool) { return 0, false }

func (b *BaseTx) ExecuteBase() error {
	if err := parser.CheckPrefix(b.Prefix); err != nil {
		return err
	}

	if bytes.Equal(b.Sender[:], emptyPublicKeyBytes[:]) {
		return ErrInvalidSender
	}

	if b.BlockID == ids.Empty {
		return ErrInvalidBlockID
	}
	return nil
}
