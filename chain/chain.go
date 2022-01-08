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
	MaxValueSize  = 1 << 10 * 128     // 128KB

	LookbackWindow     = 60
	BlockTarget        = 60
	TargetTransactions = 10 * LookbackWindow / BlockTarget // TODO: can be higher on real network

	MinDifficulty = 10 // each unit of difficulty is ~1ms and the base tx overhead is 10 units
	MinBlockCost  = 0  // in units
)

type Context struct {
	RecentBlockIDs ids.Set
	RecentTxIDs    ids.Set

	NextCost       uint64
	NextDifficulty uint64
}
