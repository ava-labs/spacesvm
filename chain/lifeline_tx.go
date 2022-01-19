// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"strconv"

	"github.com/ava-labs/spacesvm/parser"
	"github.com/ava-labs/spacesvm/tdata"
)

var _ UnsignedTransaction = &LifelineTx{}

type LifelineTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`

	// Space is the namespace for the "SpaceInfo"
	// whose owner can write and read value for the
	// specific key space.
	//
	// The space must be ^[a-z0-9]{1,256}$.
	Space string `serialize:"true" json:"space"`

	// Units is the number of [ClaimReward] to extend
	// the life of the [Space].
	Units uint64 `serialize:"true" json:"units"`
}

func (l *LifelineTx) Execute(t *TransactionContext) error {
	if l.Units == 0 {
		return ErrNonActionable
	}

	if err := parser.CheckContents(l.Space); err != nil {
		return err
	}

	g := t.Genesis
	i, has, err := GetSpaceInfo(t.Database, []byte(l.Space))
	if err != nil {
		return err
	}
	// Cannot add time to missing space
	if !has {
		return ErrSpaceMissing
	}
	// Lifeline spread across all units
	lastExpiry := i.Expiry
	i.Expiry += (g.ClaimReward * l.Units) / i.Units
	return PutSpaceInfo(t.Database, []byte(l.Space), i, lastExpiry)
}

func (l *LifelineTx) FeeUnits(g *Genesis) uint64 {
	// FeeUnits are discounted so that, all else equal, it is easier for an owner
	// to retain their space than for another to claim it.
	dSpaceNameUnits := spaceNameUnits(g, l.Space) / g.SpaceRenewalDiscount

	// The more desirable the space, the more it costs to maintain it.
	//
	// Note, this heavy base cost incentivizes users to send fewer transactions
	// to extend their space's life instead of many small ones.
	return l.LoadUnits(g) + dSpaceNameUnits*l.Units
}

func (l *LifelineTx) LoadUnits(g *Genesis) uint64 {
	return l.BaseTx.LoadUnits(g) * g.ClaimLoadMultiplier
}

func (l *LifelineTx) Copy() UnsignedTransaction {
	return &LifelineTx{
		BaseTx: l.BaseTx.Copy(),
		Space:  l.Space,
		Units:  l.Units,
	}
}

func (l *LifelineTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		l.Magic, Lifeline,
		[]tdata.Type{
			{Name: tdSpace, Type: tdString},
			{Name: tdUnits, Type: tdUint64},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdSpace:   l.Space,
			tdUnits:   strconv.FormatUint(l.Units, 10),
			tdPrice:   strconv.FormatUint(l.Price, 10),
			tdBlockID: l.BlockID.String(),
		},
	)
}

func (l *LifelineTx) Activity() *Activity {
	return &Activity{
		Typ:   Lifeline,
		Space: l.Space,
		Units: l.Units,
	}
}
