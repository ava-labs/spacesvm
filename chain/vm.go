// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
)

type VM interface {
	State() database.Database
	Mempool() Mempool

	GetBlock(ids.ID) (snowman.Block, error)
	ExecutionContext(currentTime int64, parent *StatelessBlock) (*Context, error)

	Verified(*StatelessBlock)
	Rejected(*StatelessBlock)
	Accepted(*StatelessBlock)
}
