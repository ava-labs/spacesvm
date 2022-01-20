// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"strconv"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ava-labs/spacesvm/parser"
	"github.com/ava-labs/spacesvm/tdata"
)

var _ UnsignedTransaction = &MoveTx{}

type MoveTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`

	// Space is the namespace for the "SpaceInfo"
	// whose owner can write and read value for the
	// specific key space.
	// The space must be ^[a-z0-9]{1,256}$.
	Space string `serialize:"true" json:"space"`

	// To is the recipient of the Space.
	To common.Address `serialize:"true" json:"to"`
}

func (m *MoveTx) Execute(c *TransactionContext) error {
	if err := parser.CheckContents(m.Space); err != nil {
		return err
	}

	// Must transfer to someone
	if bytes.Equal(m.To[:], zeroAddress[:]) {
		return ErrNonActionable
	}

	// This prevents someone from transferring a space to themselves.
	if bytes.Equal(m.To[:], c.Sender[:]) {
		return ErrNonActionable
	}

	// Veify space is owned by sender
	i, err := verifySpace(m.Space, c)
	if err != nil {
		return err
	}
	i.Owner = m.To

	// Update space
	if err := MoveSpaceInfo(c.Database, c.Sender, []byte(m.Space), i); err != nil {
		return err
	}
	return nil
}

func (m *MoveTx) Copy() UnsignedTransaction {
	to := make([]byte, common.AddressLength)
	copy(to, m.To[:])
	return &MoveTx{
		BaseTx: m.BaseTx.Copy(),
		Space:  m.Space,
		To:     common.BytesToAddress(to),
	}
}

func (m *MoveTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		m.Magic, Move,
		[]tdata.Type{
			{Name: tdSpace, Type: tdString},
			{Name: tdTo, Type: tdAddress},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdSpace:   m.Space,
			tdTo:      m.To.Hex(),
			tdPrice:   strconv.FormatUint(m.Price, 10),
			tdBlockID: m.BlockID.String(),
		},
	)
}

func (m *MoveTx) Activity() *Activity {
	return &Activity{
		Typ:   Move,
		Space: m.Space,
		To:    m.To.Hex(),
	}
}
