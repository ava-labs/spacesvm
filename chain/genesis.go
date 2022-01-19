// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/inconshreveable/log15"
)

const (
	LotteryRewardDivisor = 100
	MinBlockCost         = 0

	DefaultFreeClaimStorage  = 1 * units.MiB
	DefaultValueUnitSize     = 1 * units.KiB
	DefaultFreeClaimUnits    = DefaultFreeClaimStorage / DefaultValueUnitSize
	DefaultFreeClaimDuration = 60 * 60 * 24 * 30 // 30 Days

	DefaultLookbackWindow = 60
)

type Airdrop struct {
	// Address strings are hex-formatted common.Address
	Address common.Address `serialize:"true" json:"address"`
}

type CustomAllocation struct {
	// Address strings are hex-formatted common.Address
	Address common.Address `serialize:"true" json:"address"`
	Balance uint64         `serialize:"true" json:"balance"`
}

type Genesis struct {
	Magic uint64 `serialize:"true" json:"magic"`

	// Tx params
	BaseTxUnits uint64 `serialize:"true" json:"baseTxUnits"`

	// SetTx params
	ValueUnitSize       uint64 `serialize:"true" json:"valueUnitSize"`
	MaxValueSize        uint64 `serialize:"true" json:"maxValueSize"`
	ValueExpiryDiscount uint64 `serialize:"true" json:"valueExpiryDiscount"`

	// Claim Params
	ClaimLoadMultiplier         uint64 `serialize:"true" json:"claimLoadMultiplier"`
	MinClaimFee                 uint64 `serialize:"true" json:"minClaimFee"`
	SpaceDesirabilityMultiplier uint64 `serialize:"true" json:"spaceDesirabilityMultiplier"`

	// Lifeline Params
	SpaceRenewalDiscount uint64 `serialize:"true" json:"spaceRenewalDiscount"`

	// Reward Params
	ClaimReward      uint64 `serialize:"true" json:"claimReward"`
	ClaimExpiryUnits uint64 `serialize:"true" json:"claimExpiryUnits"`

	// Mining Reward (% of min required fee)
	LotteryRewardMultipler uint64 `serialize:"true" json:"lotteryRewardMultipler"` // divided by 100

	// Fee Mechanism Params
	MinPrice         uint64 `serialize:"true" json:"minPrice"`
	LookbackWindow   int64  `serialize:"true" json:"lookbackWindow"`
	TargetBlockRate  int64  `serialize:"true" json:"targetBlockRate"` // seconds
	TargetBlockSize  uint64 `serialize:"true" json:"targetBlockSize"` // units
	MaxBlockSize     uint64 `serialize:"true" json:"maxBlockSize"`    // units
	BlockCostEnabled bool   `serialize:"true" json:"blockCostEnabled"`

	// Allocations
	CustomAllocation []*CustomAllocation `serialize:"true" json:"customAllocation"`
	AirdropHash      string              `serialize:"true" json:"airdropHash"`
	AirdropUnits     uint64              `serialize:"true" json:"airdropUnits"`
}

func DefaultGenesis() *Genesis {
	return &Genesis{
		// Tx params
		BaseTxUnits: 1,

		// SetTx params
		ValueUnitSize:       DefaultValueUnitSize,
		MaxValueSize:        200 * units.KiB,
		ValueExpiryDiscount: 10,

		// Claim Params
		ClaimLoadMultiplier:         5,
		ClaimExpiryUnits:            100,
		MinClaimFee:                 100,
		SpaceDesirabilityMultiplier: 5,

		// Lifeline Params
		SpaceRenewalDiscount: 10,

		// Reward Params
		ClaimReward: DefaultFreeClaimUnits * DefaultFreeClaimDuration,

		// Lottery Reward (50% of tx.FeeUnits() * block.Price)
		LotteryRewardMultipler: 50,

		// Fee Mechanism Params
		LookbackWindow:   DefaultLookbackWindow, // 60 Seconds
		TargetBlockRate:  1,                     // 1 Block per Second
		TargetBlockSize:  225,                   // ~225KB
		MaxBlockSize:     246,                   // ~246KB -> Limited to 256KB by AvalancheGo (as of v1.7.3)
		MinPrice:         1,
		BlockCostEnabled: true,
	}
}

func (g *Genesis) StatefulBlock() *StatefulBlock {
	return &StatefulBlock{
		Price: g.MinPrice,
		Cost:  MinBlockCost,
	}
}

func (g *Genesis) Verify() error {
	if g.Magic == 0 {
		return ErrInvalidMagic
	}
	if g.TargetBlockRate == 0 {
		return ErrInvalidBlockRate
	}
	return nil
}

func (g *Genesis) Load(db database.Database, airdropData []byte) error {
	start := time.Now()
	defer func() {
		log.Debug("loaded genesis allocations", "t", time.Since(start))
	}()

	vdb := versiondb.New(db)
	if len(g.AirdropHash) > 0 {
		h := common.BytesToHash(crypto.Keccak256(airdropData)).Hex()
		if g.AirdropHash != h {
			return fmt.Errorf("expected standard allocation %s but got %s", g.AirdropHash, h)
		}

		airdrop := []*Airdrop{}
		if err := json.Unmarshal(airdropData, &airdrop); err != nil {
			return err
		}

		for _, alloc := range airdrop {
			if err := SetBalance(vdb, alloc.Address, g.AirdropUnits); err != nil {
				return fmt.Errorf("%w: addr=%s, bal=%d", err, alloc.Address, g.AirdropUnits)
			}
		}
		log.Debug(
			"applied airdrop allocation",
			"hash", h, "addrs", len(airdrop), "balance", g.AirdropUnits,
		)
	}

	// Do custom allocation last in case an address shows up in standard
	// allocation
	for _, alloc := range g.CustomAllocation {
		if err := SetBalance(vdb, alloc.Address, alloc.Balance); err != nil {
			return fmt.Errorf("%w: addr=%s, bal=%d", err, alloc.Address, alloc.Balance)
		}
		log.Debug("applied custom allocation", "addr", alloc.Address, "balance", alloc.Balance)
	}

	// Commit as a batch to improve speed
	return vdb.Commit()
}
