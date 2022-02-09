// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import "github.com/ava-labs/avalanchego/ids"

type Activity struct {
	Tmstmp int64  `serialize:"true" json:"timestamp"`
	TxID   ids.ID `serialize:"true" json:"txId"`
	Typ    string `serialize:"true" json:"type"`
	Sender string `serialize:"true" json:"sender,omitempty"` // empty when reward
	Space  string `serialize:"true" json:"space,omitempty"`
	Key    string `serialize:"true" json:"key,omitempty"`
	To     string `serialize:"true" json:"to,omitempty"` // common.Address will be 0x000 when not populated
	Units  uint64 `serialize:"true" json:"units,omitempty"`
}
