// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/ids"
)

type BaseTx struct {
	// BlkID is the ID of a block in the [lookbackWindow].
	BlockID ids.ID `serialize:"true" json:"blockId"`

	// Magic is a value defined in genesis to protect against replay attacks on
	// different VMs.
	Magic uint64 `serialize:"true" json:"magic"`

	// Price is the value per unit to spend on this transaction.
	Price uint64 `serialize:"true" json:"price"`
}

func (b *BaseTx) GetBlockID() ids.ID {
	return b.BlockID
}

func (b *BaseTx) SetBlockID(bid ids.ID) {
	b.BlockID = bid
}

func (b *BaseTx) GetMagic() uint64 {
	return b.Magic
}

func (b *BaseTx) SetMagic(magic uint64) {
	b.Magic = magic
}

func (b *BaseTx) GetPrice() uint64 {
	return b.Price
}

func (b *BaseTx) SetPrice(price uint64) {
	b.Price = price
}

func (b *BaseTx) ExecuteBase(g *Genesis) error {
	if b.BlockID == ids.Empty {
		return ErrInvalidBlockID
	}
	if b.Magic != g.Magic {
		return ErrInvalidMagic
	}
	if b.Price < g.MinPrice {
		return ErrInvalidPrice
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
	copy(blockID[:], b.BlockID[:])
	return &BaseTx{
		BlockID: blockID,
		Magic:   b.Magic,
		Price:   b.Price,
	}
}
