// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/spacesvm/parser"
	"github.com/ava-labs/spacesvm/tdata"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var _ UnsignedTransaction = &LifelineTx{}

type LifelineTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`

	// Space is the namespace for the "PrefixInfo"
	// whose owner can write and read value for the
	// specific key space.
	// The space must be ^[a-z0-9]{1,256}$.
	Space string `serialize:"true" json:"space"`

	// Units is the additional fee the sender pays to extend the life of their
	// space. The added expiry time is a function of:
	// [Units] * [LifelineInterval].
	Units uint64 `serialize:"true" json:"units"`
}

func (l *LifelineTx) Execute(t *TransactionContext) error {
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
	i.Expiry += g.LifelineUnitReward * l.Units / i.Units
	return PutSpaceInfo(t.Database, []byte(l.Space), i, lastExpiry)
}

func (l *LifelineTx) FeeUnits(g *Genesis) uint64 {
	// FeeUnits are discounted so that, all else equal, it is easier for an owner
	// to retain their space than for another to claim it.
	discountedPrefixUnits := spaceUnits(g, l.Space) / g.PrefixRenewalDiscount

	// The more desirable the space, the more it costs to maintain it.
	//
	// Note, this heavy base cost incentivizes users to send fewer transactions
	// to extend their space's life instead of many small ones.
	return l.LoadUnits(g) + discountedPrefixUnits + l.Units
}

func (l *LifelineTx) Copy() UnsignedTransaction {
	return &LifelineTx{
		BaseTx: l.BaseTx.Copy(),
		Space:  l.Space,
		Units:  l.Units,
	}
}

func (l *LifelineTx) TypedData() tdata.TypedData {
	return tdata.CreateTypedData(
		l.Magic, Lifeline,
		[]tdata.Type{
			{Name: "blockID", Type: "string"},
			{Name: "price", Type: "uint64"},
			{Name: "space", Type: "string"},
			{Name: "units", Type: "uint64"},
		},
		tdata.TypedDataMessage{
			"blockID": l.BlockID.String(),
			"price":   hexutil.EncodeUint64(l.Price),
			"space":   l.Space,
			"units":   hexutil.EncodeUint64(l.Units),
		},
	)
}
