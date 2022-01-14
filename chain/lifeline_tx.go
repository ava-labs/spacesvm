// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"

	"github.com/ava-labs/spacesvm/parser"
)

var _ UnsignedTransaction = &LifelineTx{}

type LifelineTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`

	// Units is the additional work the sender does to extend the life of their
	// prefix. The added expiry time is a function of:
	// [Units] * [LifelineInterval].
	Units uint64 `serialize:"true" json:"units"`
}

func addLife(_ *Genesis, db database.KeyValueReaderWriter, prefix []byte, reward uint64) error {
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
	i.Expiry += reward / i.Units
	return PutPrefixInfo(db, prefix, i, lastExpiry)
}

func (l *LifelineTx) Execute(t *TransactionContext) error {
	if err := parser.CheckPrefix(l.Prefix()); err != nil {
		return err
	}

	g := t.Genesis
	return addLife(g, t.Database, l.Prefix(), g.LifelineUnitReward*l.Units)
}

func (l *LifelineTx) FeeUnits(g *Genesis) uint64 {
	// FeeUnits are discounted so that, all else equal, it is easier for an owner
	// to retain their prefix than for another to claim it.
	discountedPrefixUnits := prefixUnits(g, l.Prefix()) / g.PrefixRenewalDiscount

	// The more desirable the prefix, the more it costs to maintain it.
	//
	// Note, this heavy base cost incentivizes users to send fewer transactions
	// to extend their prefix's life instead of many small ones.
	return l.LoadUnits(g) + discountedPrefixUnits + l.Units
}

func (l *LifelineTx) Copy() UnsignedTransaction {
	return &LifelineTx{
		BaseTx: l.BaseTx.Copy(),
		Units:  l.Units,
	}
}
