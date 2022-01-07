// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/utils/crypto"
)

var _ UnsignedTransaction = &ClaimTx{}

type ClaimTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`
}

func (c *ClaimTx) Execute(db database.Database, blockTime int64) error {
	// Restrict address prefix to be owned by pk
	// [33]byte prefix is reserved for pubkey
	if len(c.Prefix) == crypto.SECP256K1RPKLen && !bytes.Equal(c.Sender[:], c.Prefix) {
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

	// Anything previously at the prefix was previously removed...
	newInfo := &PrefixInfo{
		Owner:       c.Sender,
		Created:     blockTime,
		LastUpdated: blockTime,
		Expiry:      blockTime + expiryTime,
		Units:       1,
	}
	if err := PutPrefixInfo(db, c.Prefix, newInfo, -1); err != nil {
		return err
	}
	return nil
}
