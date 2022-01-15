// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

type Activity struct {
	Tmstmp int64  `serialize:"true" json:"timestamp"`
	Sender string `serialize:"true" json:"sender"`
	Typ    string `serialize:"true" json:"type"`
	Space  string `serialize:"true" json:"space,omitempty"`
	Key    string `serialize:"true" json:"key,omitempty"`
	To     string `serialize:"true" json:"to,omitempty"` // common.Address will be 0x000 when not populated
	Units  uint64 `serialize:"true" json:"units,omitempty"`
}
