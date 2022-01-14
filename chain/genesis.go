// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/inconshreveable/log15"
)

type Allocation struct {
	// Address strings are hex-formatted common.Address
	Address string `serialize:"true" json:"address"`
	Balance uint64 `serialize:"true" json:"balance"`
}

type Genesis struct {
	Magic uint64 `serialize:"true" json:"magic"`

	// Tx params
	BaseTxUnits uint64 `serialize:"true" json:"baseTxUnits"`

	// SetTx params
	ValueUnitSize uint64 `serialize:"true" json:"valueUnitSize"`
	MaxValueSize  uint64 `serialize:"true" json:"maxValueSize"`

	// Claim Params
	ClaimFeeMultiplier   uint64 `serialize:"true" json:"claimFeeMultiplier"`
	ClaimTier3Multiplier uint64 `serialize:"true" json:"claimTier3Multiplier"`
	ClaimTier2Size       uint64 `serialize:"true" json:"claimTier2Size"`
	ClaimTier2Multiplier uint64 `serialize:"true" json:"claimTier2Multiplier"`
	ClaimTier1Size       uint64 `serialize:"true" json:"claimTier1Size"`
	ClaimTier1Multiplier uint64 `serialize:"true" json:"claimTier1Multiplier"`

	// Lifeline Params
	PrefixRenewalDiscount uint64 `serialize:"true" json:"prefixRenewalDiscount"`

	// Reward Params
	ClaimReward        uint64 `serialize:"true" json:"claimReward"`
	LifelineUnitReward uint64 `serialize:"true" json:"lifelineUnitReward"`
	// TODO: replace with lottery mining
	BeneficiaryReward uint64 `serialize:"true" json:"beneficiaryReward"`

	// Fee Mechanism Params
	LookbackWindow int64  `serialize:"true" json:"lookbackWindow"`
	BlockTarget    int64  `serialize:"true" json:"blockTarget"`
	TargetUnits    uint64 `serialize:"true" json:"targetUnits"`
	MinPrice       uint64 `serialize:"true" json:"minPrice"`
	MinBlockCost   uint64 `serialize:"true" json:"minBlockCost"`

	// Allocations
	// TODO: move to a hash and use external file to avoid 1MB limit
	Allocations []*Allocation `serialize:"true" json:"allocations"`
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
		ClaimTier3Multiplier: 1,
		ClaimTier2Size:       36,
		ClaimTier2Multiplier: 5,
		ClaimTier1Size:       12,
		ClaimTier1Multiplier: 25,

		// Lifeline Params
		PrefixRenewalDiscount: 5,

		// Reward Params
		ClaimReward:        60 * 60 * 24 * 15, // 15 Days
		LifelineUnitReward: 60 * 60 * 6,       // 6 Hours Per Fee Unit (1 ms of work)
		BeneficiaryReward:  60 * 60 * 6,       // 6 Hours

		// Fee Mechanism Params
		LookbackWindow: 60,            // 60 Seconds
		BlockTarget:    1,             // 1 Block per Second
		TargetUnits:    10 * 512 * 60, // 5012 Units Per Block (~1.2MB of SetTx)
		MinPrice:       1,             // (50 for easiest claim)
		MinBlockCost:   0,             // Minimum Unit Overhead
	}
}

func (g *Genesis) StatefulBlock() *StatefulBlock {
	return &StatefulBlock{
		Price: g.MinPrice,
		Cost:  g.MinBlockCost,
	}
}

func (g *Genesis) Verify() error {
	if g.Magic == 0 {
		return ErrInvalidMagic
	}
	return nil
}

func (g *Genesis) Load(db database.KeyValueWriter) error {
	for _, alloc := range g.Allocations {
		paddr := common.HexToAddress(alloc.Address)
		if err := SetBalance(db, paddr, alloc.Balance); err != nil {
			return fmt.Errorf("%w: addr=%s, bal=%d", err, alloc.Address, alloc.Balance)
		}
		log.Debug("loaded genesis balance", "addr", paddr, "balance", alloc.Balance)
	}
	return nil
}
