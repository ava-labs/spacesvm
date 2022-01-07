// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/ids"
)

// TODO: load from genesis
const (
	MaxValueLength     = 256
	LookbackWindow     = 10
	BlockTarget        = 1
	TargetTransactions = 10 * LookbackWindow / BlockTarget // TODO: can be higher on real network

	DefaultMinDifficulty = 1 // TODO: set much higher on real network
	DefaultMinBlockCost  = 0 // in units of tx surplus
	DefaultMinExpiryTime = 30
	DefaultPruneInterval = 60
)

type Context struct {
	RecentBlockIDs ids.Set
	RecentTxIDs    ids.Set

	NextCost       uint64
	NextDifficulty uint64
	MinExpiry      uint64
}
