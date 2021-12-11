// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"

	"github.com/ava-labs/avalanchego/database"
)

var _ UnsignedTransaction = &SetTx{}

type SetTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`

	// Key is parsed from the given input, with its prefixed removed.
	// Optional for claim/lifeline transactions.
	// Non-empty to claim a key-value pair.
	Key []byte `serialize:"true" json:"key"`

	// Value is optional, and only non-empty for claim transaction with a key-value pair.
	Value []byte `serialize:"true" json:"value"`
}

func (s *SetTx) Execute(db database.Database, blockTime int64) error {
	k := append(s.Prefix, s.Key...)
	if _, _, _, err := ParseKey(k); err != nil {
		return err
	}
	if len(s.Key) == 0 {
		return ErrKeyEmpty
	}
	if len(s.Key) > MaxKeyLength {
		return ErrKeyTooBig
	}
	if len(s.Value) > MaxValueLength {
		return ErrValueTooBig
	}
	i, has, err := GetPrefixInfo(db, s.Prefix)
	if err != nil {
		return err
	}
	// Cannot set key if prefix doesn't exist
	if !has {
		return ErrPrefixMissing
	}
	// Prefix cannot be updated if not owned by modifier
	if !bytes.Equal(i.Owner[:], s.Sender[:]) {
		return ErrUnauthorized
	}
	// Prefix cannot be updated if expired
	if i.Expiry < blockTime {
		return ErrPrefixExpired
	}
	// If we are trying to delete a key, make sure it previously exists.
	if len(s.Value) > 0 {
		return s.updatePrefix(db, blockTime, i)
	}
	has, err = HasPrefixKey(db, s.Prefix, s.Key)
	if err != nil {
		return err
	}
	// Cannot delete non-existent key
	if !has {
		return ErrKeyMissing
	}
	return s.updatePrefix(db, blockTime, i)
}

func (s *SetTx) updatePrefix(db database.KeyValueWriter, blockTime int64, i *PrefixInfo) error {
	timeRemaining := (i.Expiry - i.LastUpdated) * i.Keys
	if len(s.Value) == 0 {
		i.Keys--
		if err := DeletePrefixKey(db, s.Prefix, s.Key); err != nil {
			return err
		}
	} else {
		i.Keys++
		if err := PutPrefixKey(db, s.Prefix, s.Key, s.Value); err != nil {
			return err
		}
	}
	newTimeRemaining := timeRemaining / i.Keys
	i.LastUpdated = blockTime
	i.Expiry = blockTime + newTimeRemaining
	return PutPrefixInfo(db, s.Prefix, i)
}
