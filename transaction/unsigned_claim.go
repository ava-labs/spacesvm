// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package transaction

import (
	"bytes"
	"errors"

	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/crypto/ed25519"
	"github.com/ava-labs/quarkvm/owner"
	"github.com/ava-labs/quarkvm/storage"
)

func NewClaim(sender ed25519.PublicKey, prefix []byte) Unsigned {
	return &claim{
		base: &base{
			Sender: sender,
			Prefix: prefix,
		},
	}
}

func init() {
	codec.RegisterType(&claim{})
}

var _ Unsigned = &claim{}

type claim struct {
	*base `serialize:"true"`
}

func (c *claim) Verify(s storage.Storage, blockTime int64) error {
	// Restrict address prefix to be owned by pk
	if decodedPrefix, err := formatting.Decode(formatting.CB58, string(c.Prefix)); err == nil {
		if !bytes.Equal(c.Sender.Bytes(), decodedPrefix) {
			return errors.New("public key does not match decoded prefix")
		}
	}

	has, err := s.Owner().Has(c.Prefix)
	if err != nil {
		return err
	}
	if !has {
		return ErrPrefixNotExist
	}
	v, err := s.Owner().Get(c.Prefix)
	if err != nil {
		return err
	}
	iv := new(owner.Owner)
	if _, err := codec.Unmarshal(v, iv); err != nil {
		return err
	}
	if iv.Expiry >= blockTime {
		return ErrPrefixNotExpired
	}
	return nil
}

const expiryTime = 30 // TODO: set much longer on real network

func (c *claim) Accept(s storage.Storage, blockTime int64) error {
	iv := &owner.Owner{PublicKey: c.Sender, LastUpdated: blockTime, Expiry: blockTime + expiryTime, Keys: 1}
	b, err := codec.Marshal(iv)
	if err != nil {
		return err
	}
	if err := s.Owner().Put(c.Prefix, b); err != nil {
		return err
	}
	// Remove anything that is stored in value prefix
	// return database.ClearPrefix(db, db, PrefixValueKey(c.Prefix, nil))
	return nil
}
