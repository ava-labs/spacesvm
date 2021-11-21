// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package transaction

import (
	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/owner"
	"github.com/ava-labs/quarkvm/storage"
)

func init() {
	codec.RegisterType(&lifeline{})
}

var _ Unsigned = &lifeline{}

type lifeline struct {
	*base `serialize:"true"`
}

func (kv *lifeline) Verify(s storage.Storage, blockTime int64) error {
	has, err := s.Owner().Has(kv.Prefix)
	if err != nil {
		return err
	}
	if !has {
		return ErrPrefixNotExist
	}
	// Anyone can choose to support a prefix (not just owner)
	return nil
}

func (kv *lifeline) Accept(s storage.Storage, blockTime int64) error {
	v, err := s.Owner().Get(kv.Prefix)
	if err != nil {
		return err
	}
	iv := new(owner.Owner)
	if _, err := codec.Unmarshal(v, iv); err != nil {
		return err
	}
	// If you are "in debt", lifeline only adds but doesn't reset to new
	iv.Expiry += expiryTime / iv.Keys
	b, err := codec.Marshal(iv)
	if err != nil {
		return err
	}
	return s.Owner().Put(kv.Prefix, b)
}
