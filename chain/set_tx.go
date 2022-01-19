// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"
	"strconv"

	"github.com/ava-labs/spacesvm/parser"
	"github.com/ava-labs/spacesvm/tdata"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	// 0x + hex-encoded hash
	HashLen = 66
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
	if len(s.Key) == HashLen {
		h := valueHash(s.Value)
		if s.Key != h {
			return fmt.Errorf("%w: expected %s got %x", ErrInvalidKey, h, s.Key)
		}
	}

	// Update value
	valueSize := uint64(len(s.Value))
	nvmeta := &ValueMeta{
		Size:    valueSize,
		TxID:    t.TxID,
		Updated: t.BlockTime,
	}
	v, exists, err := GetValueMeta(t.Database, []byte(s.Space), []byte(s.Key))
	if err != nil {
		return err
	}
	timeRemaining := (i.Expiry - i.Updated) * i.Units
	if exists {
		i.Units -= valueUnits(g, v.Size) / g.ValueExpiryDiscount
		nvmeta.Created = v.Created
	} else {
		nvmeta.Created = t.BlockTime
	}
	i.Units += valueUnits(g, valueSize) / g.ValueExpiryDiscount
	if err := PutSpaceKey(t.Database, []byte(s.Space), []byte(s.Key), nvmeta); err != nil {
		return err
	}
	return updateSpace(s.Space, t, timeRemaining, i)
}

func (s *SetTx) FeeUnits(g *Genesis) uint64 {
	// We don't subtract by 1 here because we want to charge extra for any
	// value-based interaction (even if it is small or a delete).
	return s.BaseTx.FeeUnits(g) + valueUnits(g, uint64(len(s.Value)))
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

func (s *SetTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		s.Magic, Set,
		[]tdata.Type{
			{Name: tdSpace, Type: tdString},
			{Name: tdKey, Type: tdString},
			{Name: tdValue, Type: tdBytes},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdSpace:   s.Space,
			tdKey:     s.Key,
			tdValue:   hexutil.Encode(s.Value),
			tdPrice:   strconv.FormatUint(s.Price, 10),
			tdBlockID: s.BlockID.String(),
		},
	)
}

func (s *SetTx) Activity() *Activity {
	return &Activity{
		Typ:   Set,
		Space: s.Space,
		Key:   s.Key,
	}
}
