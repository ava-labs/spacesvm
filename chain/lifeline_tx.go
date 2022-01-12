// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
)

var _ UnsignedTransaction = &LifelineTx{}

type LifelineTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`
}

func addLife(g *Genesis, db database.KeyValueReaderWriter, prefix []byte) error {
	i, has, err := GetPrefixInfo(db, prefix)
	if err != nil {
		return err
	}
	// Cannot add time to missing prefix
	if !has {
		return ErrPrefixMissing
	}
	// Lifeline spread across all units
	lastExpiry := i.Expiry
	i.Expiry += g.ExpiryTime / i.Units
	return PutPrefixInfo(db, prefix, i, lastExpiry)
}

func (l *LifelineTx) Execute(g *Genesis, db database.Database, blockTime uint64, _ ids.ID) error {
	return addLife(g, db, l.Prefix)
}

func (l *LifelineTx) FeeUnits(g *Genesis) uint64 {
	prefixUnits := prefixUnits(g, l.Prefix) / g.PrefixRenewalDiscount
	return l.LoadUnits(g) + prefixUnits
}

func (l *LifelineTx) Copy() UnsignedTransaction {
	return &LifelineTx{
		BaseTx: l.BaseTx.Copy(),
	}
}
