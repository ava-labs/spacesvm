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
	*BaseTx `serialize:"true"`
}

func (c *ClaimTx) Execute(db database.Database, blockTime int64) error {
	// Restrict address prefix to be owned by pk
	if len(c.Prefix) == crypto.PublicKeySize && !bytes.Equal(c.Sender[:], c.Prefix) {
		return ErrPublicKeyMismatch
	}
	previousInfo, has, err := GetPrefixInfo(db, c.Prefix)
	if err != nil {
		return err
	}
	if has && previousInfo.Expiry >= blockTime {
		return ErrPrefixNotExpired
	}
	newInfo := &PrefixInfo{Owner: c.Sender, LastUpdated: blockTime, Expiry: blockTime + expiryTime, Keys: 1}
	if err := PutPrefixInfo(db, c.Prefix, newInfo); err != nil {
		return err
	}
	// Remove anything that is stored in value prefix
	return DeleteAllPrefixKeys(db, c.Prefix)
}
