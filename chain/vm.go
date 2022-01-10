// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
)

type VM interface {
	State() database.Database
	Mempool() Mempool
	GetStatelessBlock(ids.ID) (*StatelessBlock, error)
	Beneficiary() []byte
	SetBeneficiary(prefix []byte)
	ExecutionContext(currentTime int64, parent *StatelessBlock) (*Context, error)
	Verified(*StatelessBlock)
	Rejected(*StatelessBlock)
	Accepted(*StatelessBlock)
}
