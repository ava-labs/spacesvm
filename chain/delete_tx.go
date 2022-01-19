// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"strconv"

	"github.com/ava-labs/spacesvm/parser"
	"github.com/ava-labs/spacesvm/tdata"
)

var _ UnsignedTransaction = &DeleteTx{}

type DeleteTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`

	// Space is the namespace for the "SpaceInfo"
	// whose owner can write and read value for the
	// specific key space.
	// The space must be ^[a-z0-9]{1,256}$.
	Space string `serialize:"true" json:"space"`

	// Key is parsed from the given input, with its space removed.
	Key string `serialize:"true" json:"key"`
}

func (d *DeleteTx) Execute(t *TransactionContext) error {
	g := t.Genesis
	if err := parser.CheckContents(d.Space); err != nil {
		return err
	}
	if err := parser.CheckContents(d.Key); err != nil {
		return err
	}

	// Verify space is owned by sender
	i, err := verifySpace(d.Space, t)
	if err != nil {
		return err
	}

	// Delete value
	v, exists, err := GetValueMeta(t.Database, []byte(d.Space), []byte(d.Key))
	if err != nil {
		return err
	}
	if !exists {
		return ErrKeyMissing
	}
	timeRemaining := (i.Expiry - i.Updated) * i.Units
	i.Units -= valueUnits(g, v.Size) / g.ValueExpiryDiscount
	if err := DeleteSpaceKey(t.Database, []byte(d.Space), []byte(d.Key)); err != nil {
		return err
	}
	return updateSpace(d.Space, t, timeRemaining, i)
}

func (d *DeleteTx) Copy() UnsignedTransaction {
	return &DeleteTx{
		BaseTx: d.BaseTx.Copy(),
		Space:  d.Space,
		Key:    d.Key,
	}
}

func (d *DeleteTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		d.Magic, Delete,
		[]tdata.Type{
			{Name: tdSpace, Type: tdString},
			{Name: tdKey, Type: tdString},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdSpace:   d.Space,
			tdKey:     d.Key,
			tdPrice:   strconv.FormatUint(d.Price, 10),
			tdBlockID: d.BlockID.String(),
		},
	)
}

func (d *DeleteTx) Activity() *Activity {
	return &Activity{
		Typ:   Delete,
		Space: d.Space,
		Key:   d.Key,
	}
}
