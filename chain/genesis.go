// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

type Genesis struct {
	MinDifficulty uint64 `serialize:"true" json:"minDifficulty"`
	MinBlockCost  uint64 `serialize:"true" json:"minBlockCost"`

	// MinExpiry is the minimum number of seconds allowed
	// to expire prefix since its block time.
	MinExpiry uint64 `serialize:"true" json:"minExpiry"`

	// PruneInterval is the prune interval in seconds.
	PruneInterval uint64 `serialize:"true" json:"pruneInterval"`
}
