// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ava-labs/quarkvm/parser"
)

var _ UnsignedTransaction = &ClaimTx{}

type ClaimTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`
}

func (c *ClaimTx) Execute(t *TransactionContext) error {
	// Restrict address prefix to be owned by address
	if len(c.Prefix()) == common.AddressLength && !bytes.Equal(t.Sender[:], c.Prefix()) {
		return ErrAddressMismatch
	}

	// Prefix keys only exist if they are still valid
	exists, err := HasPrefix(t.Database, c.Prefix())
	if err != nil {
		return err
	}
	if exists {
		return ErrPrefixNotExpired
	}

	// Anything previously at the prefix was previously removed...
	newInfo := &PrefixInfo{
		Owner:       t.Sender,
		Created:     t.BlockTime,
		LastUpdated: t.BlockTime,
		Expiry:      t.BlockTime + t.Genesis.ClaimReward,
		Units:       1,
	}
	if err := PutPrefixInfo(t.Database, c.Prefix(), newInfo, 0); err != nil {
		return err
	}
	return nil
}

// [prefixUnits] requires the caller to pay more to get prefixes of
// a shorter length because they are more desirable. This creates a "lottery"
// mechanism where the people that spend the most mining power will win the
// prefix.
//
// [prefixUnits] should only be called on a prefix that is valid
func prefixUnits(g *Genesis, p []byte) uint64 {
	desirability := uint64(parser.MaxKeySize - len(p))
	if uint64(len(p)) > g.ClaimTier2Size {
		return desirability * g.ClaimTier3Multiplier
	}
	if uint64(len(p)) > g.ClaimTier1Size {
		return desirability * g.ClaimTier2Multiplier
	}
	return desirability * g.ClaimTier1Multiplier
}

func (c *ClaimTx) FeeUnits(g *Genesis) uint64 {
	return c.LoadUnits(g) + prefixUnits(g, c.Prefix())
}

func (c *ClaimTx) LoadUnits(g *Genesis) uint64 {
	return c.BaseTx.LoadUnits(g) * g.ClaimFeeMultiplier
}

func (c *ClaimTx) Copy() UnsignedTransaction {
	return &ClaimTx{
		BaseTx: c.BaseTx.Copy(),
	}
}
