// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"

	"github.com/ava-labs/spacesvm/parser"
	"github.com/ava-labs/spacesvm/tdata"
)

const (
	// 0x + hex-encoded hash
	hashLen = 66
)

var _ UnsignedTransaction = &SetTx{}

type SetTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`

	// Space is the namespace for the "SpaceInfo"
	// whose owner can write and read value for the
	// specific key space.
	// The space must be ^[a-z0-9]{1,256}$.
	Space string `serialize:"true" json:"space"`

	// Key is parsed from the given input, with its space removed.
	Key string `serialize:"true" json:"key"`

	// Value is written as the key-value pair to the storage. If a previous value
	// exists, it is overwritten.
	Value []byte `serialize:"true" json:"value"`
}

func (s *SetTx) Execute(t *TransactionContext) error {
	g := t.Genesis
	if err := parser.CheckContents(s.Space); err != nil {
		return err
	}
	if err := parser.CheckContents(s.Key); err != nil {
		return err
	}
	switch {
	case len(s.Value) == 0:
		return ErrValueEmpty
	case uint64(len(s.Value)) > g.MaxValueSize:
		return ErrValueTooBig
	}

	// Verify space is owned by sender
	i, err := verifySpace(s.Space, t)
	if err != nil {
		return err
	}

	// If Key is equal to hash length, ensure it is equal to the hash of the
	// value
	if len(s.Key) == hashLen {
		h := valueHash(s.Value)
		if s.Key != h {
			return fmt.Errorf("%w: expected %s got %x", ErrInvalidKey, h, s.Key)
		}
	}

	// Update value
	v, exists, err := GetValue(t.Database, []byte(s.Space), []byte(s.Key))
	if err != nil {
		return err
	}
	timeRemaining := (i.Expiry - i.LastUpdated) * i.Units
	if exists {
		i.Units -= valueUnits(g, v)
	}
	i.Units += valueUnits(g, s.Value)
	if err := PutSpaceKey(t.Database, []byte(s.Space), []byte(s.Key), t.TxID[:]); err != nil {
		return err
	}
	return updateSpace(s.Space, t, timeRemaining, i)
}

func (s *SetTx) FeeUnits(g *Genesis) uint64 {
	// We don't subtract by 1 here because we want to charge extra for any
	// value-based interaction (even if it is small or a delete).
	return s.BaseTx.FeeUnits(g) + valueUnits(g, s.Value)
}

func (s *SetTx) LoadUnits(g *Genesis) uint64 {
	return s.FeeUnits(g)
}

func (s *SetTx) Copy() UnsignedTransaction {
	value := make([]byte, len(s.Value))
	copy(value, s.Value)
	return &SetTx{
		BaseTx: s.BaseTx.Copy(),
		Space:  s.Space,
		Key:    s.Key,
		Value:  value,
	}
}

func (s *SetTx) TypedData() tdata.TypedData {
	return tdata.CreateTypedData(
		s.Magic, Set,
		[]tdata.Type{
			{Name: "blockID", Type: "string"},
			{Name: "price", Type: "uint64"},
			{Name: "space", Type: "string"},
			{Name: "key", Type: "string"},
			{Name: "value", Type: "bytes"},
		},
		tdata.TypedDataMessage{
			"blockID": s.BlockID.String(),
			"price":   s.Price,
			"space":   s.Space,
			"key":     s.Key,
			"value":   s.Value,
		},
	)
}
