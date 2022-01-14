// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
)

type PrefixInfo struct {
	Owner       common.Address `serialize:"true" json:"owner"`
	Created     uint64         `serialize:"true" json:"created"`
	LastUpdated uint64         `serialize:"true" json:"lastUpdated"`
	Expiry      uint64         `serialize:"true" json:"expiry"`
	Units       uint64         `serialize:"true" json:"units"` // decays faster the more units you have

	RawPrefix ids.ShortID `serialize:"true" json:"rawPrefix"`
}
