// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/utils/units"
)

type Genesis struct {
	// Tx params
	BaseTxUnits uint64 `serialize:"true" json:"baseTxUnits"`

	// SetTx params
	ValueUnitSize uint64 `serialize:"true" json:"valueUnitSize"`
	MaxValueSize  uint64 `serialize:"true" json:"maxValueSize"`

	// Claim Params
	ClaimFeeMultiplier   uint64 `serialize:"true" json:"claimFeeMultiplier"`
	ClaimReward          uint64 `serialize:"true" json:"claimReward"`
	ClaimTier3Multiplier uint64 `serialize:"true" json:"claimTier3Multiplier"`
	ClaimTier2Size       uint64 `serialize:"true" json:"claimTier2Size"`
	ClaimTier2Multiplier uint64 `serialize:"true" json:"claimTier2Multiplier"`
	ClaimTier1Size       uint64 `serialize:"true" json:"claimTier1Size"`
	ClaimTier1Multiplier uint64 `serialize:"true" json:"claimTier1Multiplier"`

	// Lifeline Params
	PrefixRenewalDiscount uint64 `serialize:"true" json:"prefixRenewalDiscount"`
	BeneficiaryReward     uint64 `serialize:"true" json:"beneficiaryReward"`

	// Fee Mechanism Params
	LookbackWindow int64  `serialize:"true" json:"lookbackWindow"`
	BlockTarget    int64  `serialize:"true" json:"blockTarget"`
	TargetUnits    uint64 `serialize:"true" json:"targetUnits"`
	MinDifficulty  uint64 `serialize:"true" json:"minDifficulty"`
	MinBlockCost   uint64 `serialize:"true" json:"minBlockCost"`
}

func DefaultGenesis() *Genesis {
	return &Genesis{
		// Tx params
		BaseTxUnits: 10,

		// SetTx params
		ValueUnitSize: 256,             // 256B
		MaxValueSize:  128 * units.KiB, // (500 Units)

		// Claim Params
		ClaimFeeMultiplier:   5,
		ClaimReward:          60 * 60 * 24 * 30, // 30 Days
		ClaimTier3Multiplier: 1,
		ClaimTier2Size:       36,
		ClaimTier2Multiplier: 5,
		ClaimTier1Size:       12,
		ClaimTier1Multiplier: 25,

		// Lifeline Params
		PrefixRenewalDiscount: 5,
		BeneficiaryReward:     60 * 60 * 6, // 6 Hours

		// Fee Mechanism Params
		LookbackWindow: 60,            // 60 Seconds
		BlockTarget:    1,             // 1 Block per Second
		TargetUnits:    10 * 512 * 60, // 5012 Units Per Block (~1.2MB of SetTx)
		MinDifficulty:  100,           // ~100ms per unit (~5s for easiest claim)
		MinBlockCost:   1,             // Minimum Unit Overhead
	}
}

func (g *Genesis) StatefulBlock() *StatefulBlock {
	return &StatefulBlock{
		Difficulty: g.MinDifficulty,
		Cost:       g.MinBlockCost,
	}
}
