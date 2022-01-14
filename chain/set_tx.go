// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/quarkvm/parser"
)

const IDLen = 32

var _ UnsignedTransaction = &SetTx{}

type SetTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`

	// Key is parsed from the given input, with its prefix removed.
	// TODO: change to string
	Key []byte `serialize:"true" json:"key"`
	// Value is empty if and only if set transaction is issued for the delete.
	// If non-empty, the transaction writes the key-value pair to the storage.
	// If empty, the transaction deletes the value for the "prefix/key".
	Value []byte `serialize:"true" json:"value"`
}

func (s *SetTx) Execute(t *TransactionContext) error {
	if err := parser.CheckPrefix(s.Prefix()); err != nil {
		return err
	}

	g := t.Genesis
	// assume prefix is already validated via "BaseTx"
	if err := parser.CheckKey(s.Key); err != nil {
		return err
	}
	if uint64(len(s.Value)) > g.MaxValueSize {
		return ErrValueTooBig
	}

	i, has, err := GetPrefixInfo(t.Database, s.Prefix())
	if err != nil {
		return err
	}
	// Cannot set key if prefix doesn't exist
	if !has {
		return ErrPrefixMissing
	}
	// Prefix cannot be updated if not owned by modifier
	if !bytes.Equal(i.Owner[:], t.Sender[:]) {
		return ErrUnauthorized
	}
	// Prefix cannot be updated if expired
	if i.Expiry < t.BlockTime {
		return ErrPrefixExpired
	}
	// TODO: hash could contain `/` (just hex encode it)
	// // If Key is equal to hash length, ensure it is equal to the hash of the
	// // value
	// if len(s.Key) == IDLen && len(s.Value) > 0 {
	// 	id, err := ids.ToID(hashing.ComputeHash256(s.Value))
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if !bytes.Equal(s.Key, id[:]) {
	// 		return fmt.Errorf("%w: expected %x got %x", ErrInvalidKey, id[:], s.Key)
	// 	}
	// }
	return s.updatePrefix(g, t.Database, t.BlockTime, t.TxID, i)
}

func (s *SetTx) updatePrefix(g *Genesis, db database.Database, blockTime uint64, txID ids.ID, i *PrefixInfo) error {
	v, exists, err := GetValue(db, s.Prefix(), s.Key)
	if err != nil {
		return err
	}

	timeRemaining := (i.Expiry - i.LastUpdated) * i.Units
	if len(s.Value) == 0 { //nolint:nestif
		if !exists {
			return ErrKeyMissing
		}
		i.Units -= valueUnits(g, v)
		if err := DeletePrefixKey(db, s.Prefix(), s.Key); err != nil {
			return err
		}
	} else {
		if exists {
			i.Units -= valueUnits(g, v)
		}
		i.Units += valueUnits(g, s.Value)
		if err := PutPrefixKey(db, s.Prefix(), s.Key, txID[:]); err != nil {
			return err
		}
	}
	newTimeRemaining := timeRemaining / i.Units
	i.LastUpdated = blockTime
	lastExpiry := i.Expiry
	i.Expiry = blockTime + newTimeRemaining
	return PutPrefixInfo(db, s.Prefix(), i, lastExpiry)
}

func valueUnits(g *Genesis, b []byte) uint64 {
	return uint64(len(b))/g.ValueUnitSize + 1
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
	key := make([]byte, len(s.Key))
	copy(key, s.Key)
	value := make([]byte, len(s.Value))
	copy(value, s.Value)
	return &SetTx{
		BaseTx: s.BaseTx.Copy(),
		Key:    key,
		Value:  value,
	}
}
