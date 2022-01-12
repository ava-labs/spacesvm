// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"time"

	"github.com/ava-labs/avalanchego/ids"
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
	ExpiryTime           uint64 `serialize:"true" json:"expiryTime"`
	ClaimTier3Multiplier uint64 `serialize:"true" json:"claimTier3Multiplier"`
	ClaimTier2Size       uint64 `serialize:"true" json:"claimTier2Size"`
	ClaimTier2Multiplier uint64 `serialize:"true" json:"claimTier2Multiplier"`
	ClaimTier1Size       uint64 `serialize:"true" json:"claimTier1Size"`
	ClaimTier1Multiplier uint64 `serialize:"true" json:"claimTier1Multiplier"`

	// Lifeline Params
	PrefixRenewalDiscount uint64 `serialize:"true" json:"prefixRenewalDiscount"`

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
		ExpiryTime:           60 * 60 * 24 * 30, // 30 Days
		ClaimTier3Multiplier: 1,
		ClaimTier2Size:       36,
		ClaimTier2Multiplier: 5,
		ClaimTier1Size:       12,
		ClaimTier1Multiplier: 25,

		// Lifeline Params
		PrefixRenewalDiscount: 5,

		// Fee Mechanism Params
		LookbackWindow: 60,            // 60 Seconds
		BlockTarget:    1,             // 1 Block per Second
		TargetUnits:    10 * 512 * 60, // 5012 Units Per Block (~1.2MB of SetTx)
		MinDifficulty:  100,           // ~100ms per unit (~5s for easiest claim)
		MinBlockCost:   1,             // Minimum Unit Overhead
	}
}

func (b *StatelessBlock) VerifyGenesis() (*Genesis, error) {
	if b.Prnt != ids.Empty {
		return nil, ErrInvalidGenesisParent
	}
	if b.Hght != 0 {
		return nil, ErrInvalidGenesisHeight
	}
	if b.Tmstmp == 0 || time.Now().Unix()-b.Tmstmp < 0 {
		return nil, ErrInvalidGenesisTimestamp
	}
	if b.Genesis == nil {
		return nil, ErrMissingGenesis
	}
	if b.Difficulty != b.genesis.MinDifficulty {
		return nil, ErrInvalidGenesisDifficulty
	}
	if b.Cost != b.genesis.MinBlockCost {
		return nil, ErrInvalidGenesisCost
	}
	if len(b.Txs) > 0 {
		return nil, ErrInvalidGenesisTxs
	}
	if len(b.Beneficiary) > 0 {
		return nil, ErrInvalidGenesisBeneficiary
	}
	return &b.genesis, nil
}
