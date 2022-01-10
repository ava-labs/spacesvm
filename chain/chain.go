// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/units"
)

// TODO: load from genesis
const (
	// Tx params
	BaseTxUnits = 10

	// SetTx params
	ValueUnitSize = 256             // 256B
	MaxValueSize  = 128 * units.KiB // (500 Units)

	// Claim Params
	ClaimFeeMultiplier   = 5
	ExpiryTime           = 60 * 60 * 24 * 30 // 30 Days
	ClaimTier3Multiplier = 1
	ClaimTier2Size       = 36
	ClaimTier2Multiplier = 5
	ClaimTier1Size       = 12
	ClaimTier1Multiplier = 25

	// Lifeline Params
	PrefixRenewalDiscount = 5

	// Fee Mechanism Params
	LookbackWindow = 60                                               // 60 Seconds
	BlockTarget    = 1                                                // 1 Block per Second
	TargetUnits    = BaseTxUnits * 512 * LookbackWindow / BlockTarget // 5012 Units Per Block (~1.2MB of SetTx)
	MinDifficulty  = 50                                               // ~50ms per unit (~2.5s for easiest claim)
	MinBlockCost   = 1                                                // Minimum Unit Overhead
)

type Context struct {
	RecentBlockIDs ids.Set
	RecentTxIDs    ids.Set
	RecentUnits    uint64

	Difficulties []uint64
	Costs        []uint64

	NextCost       uint64
	NextDifficulty uint64
}
