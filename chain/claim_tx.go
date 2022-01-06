// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"

	"github.com/ava-labs/avalanchego/database"

	"github.com/ava-labs/quarkvm/crypto"
)

var _ UnsignedTransaction = &ClaimTx{}

type ClaimTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`
}

func (c *ClaimTx) Execute(db database.Database, blockTime int64) error {
	// Restrict address prefix to be owned by pk
	// [32]byte prefix is reserved for pubkey
	if len(c.Prefix) == crypto.PublicKeySize && !bytes.Equal(c.Sender[:], c.Prefix) {
		return ErrPublicKeyMismatch
	}

	// Prefix keys only exist if they are still valid
	exists, err := HasPrefix(db, c.Prefix)
	if err != nil {
		return err
	}
	if exists {
		return ErrPrefixNotExpired
	}

	// Anything previously at the index was previously removed
	rawPrefix, err := RawPrefix(c.Prefix, blockTime)
	newInfo := &PrefixInfo{
		Owner:       c.Sender,
		RawPrefix:   rawPrefix,
		LastUpdated: blockTime,
		Expiry:      blockTime + expiryTime,
		Keys:        1,
	}
	if err := PutPrefixInfo(db, c.Prefix, newInfo); err != nil {
		return err
	}
	return nil
}
