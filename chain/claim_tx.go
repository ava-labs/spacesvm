// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ava-labs/spacesvm/parser"
	"github.com/ava-labs/spacesvm/tdata"
)

var _ UnsignedTransaction = &ClaimTx{}

const (
	// 0x + hex-encoded addr
	hexAddressLen = 2 + common.AddressLength*2
)

type ClaimTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`

	// Space is the namespace for the "SpaceInfo"
	// whose owner can write and read value for the
	// specific key space.
	// The space must be ^[a-z0-9]{1,256}$.
	Space string `serialize:"true" json:"space"`
}

func (c *ClaimTx) Execute(t *TransactionContext) error {
	if err := parser.CheckContents(c.Space); err != nil {
		return err
	}

	// Restrict address prefix to be owned by address
	if len(c.Space) == hexAddressLen && strings.ToLower(t.Sender.Hex()) != c.Space {
		return ErrAddressMismatch
	}

	// Space keys only exist if they are still valid
	exists, err := HasSpace(t.Database, []byte(c.Space))
	if err != nil {
		return err
	}
	if exists {
		return ErrSpaceNotExpired
	}

	// Anything previously at the space was previously removed...
	newInfo := &SpaceInfo{
		Owner:       t.Sender,
		Created:     t.BlockTime,
		LastUpdated: t.BlockTime,
		Expiry:      t.BlockTime + t.Genesis.ClaimReward,
		Units:       1,
	}
	if err := PutSpaceInfo(t.Database, []byte(c.Space), newInfo, 0); err != nil {
		return err
	}
	return nil
}

// [spaceUnits] requires the caller to pay more to get spaces of
// a shorter length because they are more desirable. This creates a "lottery"
// mechanism where the people that spend the most mining power will win the
// space.
//
// [spaceUnits] should only be called on a space that is valid
func spaceUnits(g *Genesis, s string) uint64 {
	desirability := uint64(parser.MaxIdentifierSize - len(s))
	if uint64(len(s)) > g.ClaimTier2Size {
		return desirability * g.ClaimTier3Multiplier
	}
	if uint64(len(s)) > g.ClaimTier1Size {
		return desirability * g.ClaimTier2Multiplier
	}
	return desirability * g.ClaimTier1Multiplier
}

func (c *ClaimTx) FeeUnits(g *Genesis) uint64 {
	return c.LoadUnits(g) + spaceUnits(g, c.Space)
}

func (c *ClaimTx) LoadUnits(g *Genesis) uint64 {
	return c.BaseTx.LoadUnits(g) * g.ClaimFeeMultiplier
}

func (c *ClaimTx) Copy() UnsignedTransaction {
	return &ClaimTx{
		BaseTx: c.BaseTx.Copy(),
		Space:  c.Space,
	}
}

func (c *ClaimTx) TypedData() tdata.TypedData {
	return tdata.CreateTypedData(
		c.Magic, "ClaimTx",
		[]tdata.Type{
			{Name: "blockID", Type: "string"},
			{Name: "price", Type: "uint64"},
			{Name: "space", Type: "string"},
		},
		tdata.TypedDataMessage{
			"blockID": c.BlockID.String(),
			"price":   c.Price,
			"space":   c.Space,
		},
	)
}
