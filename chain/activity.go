// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import "github.com/ethereum/go-ethereum/common"

type Activity struct {
	Tmstmp int64          `serialize:"true" json:"timestamp"`
	Typ    string         `serialize:"true" json:"type"`
	Space  string         `serialize:"true" json:"space,omitempty"`
	Key    string         `serialize:"true" json:"key,omitempty"`
	To     common.Address `serialize:"true" json:"to,omitempty"`
	Units  uint64         `serialize:"true" json:"units,omitempty"`
}
