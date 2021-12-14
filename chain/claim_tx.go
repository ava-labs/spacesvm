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

	prevInfo, infoExists, err := GetPrefixInfo(db, c.Prefix)
	if err != nil {
		return err
	}
	if infoExists && prevInfo.Expiry >= blockTime {
		return ErrPrefixNotExpired
	}

	// every successful "claim" deletes the existing keys
	// whether "c.Sender" is same as or different than "prevInfo.Owner"
	// now write with either prefix expired or new prefix owner
	newInfo := &PrefixInfo{
		Owner:       c.Sender,
		LastUpdated: blockTime,
		Expiry:      blockTime + expiryTime,
		Keys:        1,
	}
	if err := PutPrefixInfo(db, c.Prefix, newInfo); err != nil {
		return err
	}

	// Remove anything that is stored in value prefix
	// overwrite even if claimed by the same owner
	// TODO(patrick-ogrady): free things async for faster block verification loops
	// e.g., lazily free what is said to be freed in the block?
	return DeleteAllPrefixKeys(db, c.Prefix)
}
