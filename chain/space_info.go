// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
)

type SpaceInfo struct {
	Owner   common.Address `serialize:"true" json:"owner"`
	Created uint64         `serialize:"true" json:"created"`
	Updated uint64         `serialize:"true" json:"updated"`
	Expiry  uint64         `serialize:"true" json:"expiry"`
	Units   uint64         `serialize:"true" json:"units"` // decays faster the more units you have

	RawSpace ids.ShortID `serialize:"true" json:"rawSpace"`
}
