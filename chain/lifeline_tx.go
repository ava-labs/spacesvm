// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"

	"github.com/ava-labs/avalanchego/database"
)

var _ UnsignedTransaction = &LifelineTx{}

type LifelineTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`
}

func (l *LifelineTx) Execute(db database.Database, blockTime int64) error {
	curInfo, has, err := GetPrefixInfo(db, l.Prefix)
	if err != nil {
		return err
	}
	// Cannot add lifeline to missing prefix
	if !has {
		return ErrPrefixMissing
	}
	// Prefix cannot be updated if not owned by modifier
	if !bytes.Equal(curInfo.Owner[:], l.Sender[:]) {
		return ErrUnauthorized
	}
	// If you are "in debt", lifeline only adds but doesn't reset to new
	lastExpiry := curInfo.Expiry
	curInfo.Expiry += expiryTime / curInfo.Keys
	return PutPrefixInfo(db, l.Prefix, curInfo, lastExpiry)
}
