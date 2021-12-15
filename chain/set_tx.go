// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/quarkvm/storage"
)

var _ UnsignedTransaction = &SetTx{}

type SetTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`

	// Key is parsed from the given input, with its prefix removed.
	Key []byte `serialize:"true" json:"key"`
	// Value is empty if and only if set transaction is issued for the delete.
	// If non-empty, the transaction writes the key-value pair to the storage.
	// If empty, the transaction deletes the value for the "prefix/key".
	Value []byte `serialize:"true" json:"value"`

	// TODO: support range deletes?
}

func (s *SetTx) Execute(db database.Database, blockTime int64) error {
	// assume prefix is already validated via "BaseTx"
	if err := storage.CheckKey(s.Key); err != nil {
		return err
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
