// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/quarkvm/crypto"
)

type BaseTx struct {
	Sender   [crypto.PublicKeySize]byte `serialize:"true" json:"sender"`
	Graffiti uint64                     `serialize:"true" json:"graffiti"`
	BlockID  ids.ID                     `serialize:"true" json:"blockId"`

	// Prefix is the namespace for the "PrefixInfo"
	// whose owner can write and read value for the
	// specific key space.
	// Assume the client pre-processes the inputs so that
	// all prefix must have the delimiter '/' as suffix.
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

func (b *BaseTx) GetSender() [crypto.PublicKeySize]byte {
	return b.Sender
}

func (b *BaseTx) ExecuteBase() error {
	if _, _, _, err := ParseKey(b.Prefix); err != nil {
		return err
	}

	// "len(b.Sender) == 0" does not check zeroed [32]byte array
	if crypto.IsEmptyPublicKey(b.Sender[:]) {
		return ErrInvalidSender
	}

	if b.BlockID == ids.Empty {
		return ErrInvalidBlockID
	}
	return nil
}
