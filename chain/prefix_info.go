// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/quarkvm/crypto"
)

type PrefixInfo struct {
	Owner       [crypto.PublicKeySize]byte `serialize:"true" json:"owner"`
	RawPrefix   ids.ShortID                `serialize:"true" json:"-"`
	LastUpdated int64                      `serialize:"true" json:"lastUpdated"`
	Expiry      int64                      `serialize:"true" json:"expiry"`
	Keys        int64                      `serialize:"true" json:"keys"` // decays faster the more keys you have
}
