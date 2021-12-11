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
	if crypto.IsEmptyPublicKey(c.Sender[:]) {
		return ErrInvalidSender
	}

	prevInfo, infoExists, err := GetPrefixInfo(db, c.Prefix)
	if err != nil {
		return err
	}
	if infoExists && prevInfo.Expiry >= blockTime {
		return ErrPrefixNotExpired
	}

	if infoExists && !bytes.Equal(prevInfo.Owner[:], c.Sender[:]) {
		// only clean up when claimed by different owner
		// if same owner, don't overwrite
		if err := DeleteAllPrefixKeys(db, c.Prefix); err != nil {
			return err
		}
	}

	// either prefix expired or new prefix owner
	newInfo := &PrefixInfo{Owner: c.Sender, LastUpdated: blockTime, Expiry: blockTime + expiryTime}
	switch {
	case infoExists:
		// use previous info's keys
		newInfo.Keys = prevInfo.Keys
	case !infoExists:
		// new claim, prefix was never written
		newInfo.Keys = 1
	}
	return PutPrefixInfo(db, c.Prefix, newInfo)
}
