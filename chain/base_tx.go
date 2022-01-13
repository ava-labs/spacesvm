// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/quarkvm/parser"
)

type BaseTx struct {
	BlkID ids.ID `serialize:"true" json:"blockId"`

	// Prefix is the namespace for the "PrefixInfo"
	// whose owner can write and read value for the
	// specific key space.
	// The prefix must not have the delimiter '/' as suffix.
	// Otherwise, the verification will fail.
	Pfx []byte `serialize:"true" json:"prefix"`

	// Magic is a value defined in genesis to protect against replay attacks on
	// different VMs.
	Mgc uint64 `serialize:"true" json:"magic"`
}

func (b *BaseTx) BlockID() ids.ID {
	return b.BlkID
}

func (b *BaseTx) Prefix() []byte {
	return b.Pfx
}

func (b *BaseTx) Magic() uint64 {
	return b.Mgc
}

func (b *BaseTx) ExecuteBase(g *Genesis) error {
	if err := parser.CheckPrefix(b.Pfx); err != nil {
		return err
	}
	if b.BlkID == ids.Empty {
		return ErrInvalidBlockID
	}
	if g.Magic != b.Mgc {
		return ErrInvalidMagic
	}
	return nil
}

func (b *BaseTx) FeeUnits(g *Genesis) uint64 {
	return g.BaseTxUnits
}

func (b *BaseTx) LoadUnits(g *Genesis) uint64 {
	return b.FeeUnits(g)
}

func (b *BaseTx) Copy() *BaseTx {
	blockID := ids.ID{}
	copy(blockID[:], b.BlkID[:])
	prefix := make([]byte, len(b.Pfx))
	copy(prefix, b.Pfx)
	return &BaseTx{
		BlkID: blockID,
		Pfx:   prefix,
		Mgc:   b.Mgc,
	}
}
