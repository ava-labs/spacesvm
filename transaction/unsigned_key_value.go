// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package transaction

import (
	"errors"

	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/owner"
	"github.com/ava-labs/quarkvm/storage"
)

func init() {
	codec.RegisterType(&keyValue{})
}

var _ Unsigned = &keyValue{}

type keyValue struct {
	*base `serialize:"true"`
	Key   []byte `serialize:"true"`
	Value []byte `serialize:"true"`
}

func (kv *keyValue) Verify(s storage.Storage, blockTime int64) error {
	if len(kv.Key) > maxKeyLength || len(kv.Key) == 0 {
		return errors.New("invalid key length")
	}
	if len(kv.Value) > maxKeyLength {
		return errors.New("invalid value length")
	}

	has, err := s.Owner().Has(kv.Prefix)
	if err != nil {
		return err
	}
	if !has {
		return ErrPrefixNotExist
	}
	v, err := s.Owner().Get(kv.Prefix)
	if err != nil {
		return err
	}
	iv := new(owner.Owner)
	if _, err := codec.Unmarshal(v, iv); err != nil {
		return err
	}
	if iv.PublicKey != kv.Sender {
		return ErrPrefixOwnerMismatch
	}
	if iv.Expiry < blockTime {
		return ErrPrefixExpired
	}

	if len(kv.Value) == 0 {
		return nil
	}

	has, err = s.Key().Has(kv.Key)
	if err != nil {
		return err
	}
	if !has {
		return ErrKeyNotExist
	}
	return nil
}

func (kv *keyValue) Accept(s storage.Storage, blockTime int64) error {
	v, err := s.Owner().Get(kv.Prefix)
	if err != nil {
		return err
	}
	iv := new(owner.Owner)
	if _, err := codec.Unmarshal(v, iv); err != nil {
		return err
	}

	timeRemaining := (iv.Expiry - iv.LastUpdated) * iv.Keys
	if len(kv.Value) == 0 {
		iv.Keys--
		if err := s.Key().Delete(kv.Key); err != nil {
			return err
		}
	} else {
		iv.Keys++
		if err := s.Key().Put(kv.Key, kv.Value); err != nil {
			return err
		}
	}
	newTimeRemaining := timeRemaining / iv.Keys
	iv.LastUpdated = blockTime
	iv.Expiry = blockTime + newTimeRemaining
	b, err := codec.Marshal(iv)
	if err != nil {
		return err
	}
	return s.Owner().Put(kv.Prefix, b)
}
