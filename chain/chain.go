// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/ids"
)

// TODO: load from genesis
const (
	ExpiryTime    = 60 * 60 * 24 * 30 // 30 Days
	ValueUnitSize = 256               // 256B
	MaxValueSize  = 1 << 10 * 128     // 128KB (500 Units)
	BaseTxUnits   = 10

	LookbackWindow = 60                                               // 60 Seconds
	BlockTarget    = 60                                               // 60 Blocks per Lookback Window
	TargetUnits    = BaseTxUnits * 512 * LookbackWindow / BlockTarget // 512 Units Per Block

	MinDifficulty = 10 // ~10ms per unit (100 ms for claim)
	MinBlockCost  = 1  // Minimum Unit Overhead
)

type Context struct {
	RecentBlockIDs ids.Set
	RecentTxIDs    ids.Set
	RecentUnits    uint64

	NextCost       uint64
	NextDifficulty uint64
}
