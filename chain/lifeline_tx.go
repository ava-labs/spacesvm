// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
)

var _ UnsignedTransaction = &LifelineTx{}

type LifelineTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`
}

func (l *LifelineTx) Execute(db database.Database, blockTime uint64) error {
	i, has, err := GetPrefixInfo(db, l.Prefix)
	if err != nil {
		return err
	}
	// Cannot add lifeline to missing prefix
	if !has {
		return ErrPrefixMissing
	}
	// Lifeline spread across all units
	lastExpiry := i.Expiry
	i.Expiry += ExpiryTime / i.Units
	return PutPrefixInfo(db, l.Prefix, i, lastExpiry)
}
