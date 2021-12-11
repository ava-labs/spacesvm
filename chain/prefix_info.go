// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/quarkvm/crypto"
)

type PrefixInfo struct {
	Owner       [crypto.PublicKeySize]byte `serialize:"true"`
	LastUpdated int64                      `serialize:"true"`
	Expiry      int64                      `serialize:"true"`
	Keys        int64                      `serialize:"true"` // decays faster the more keys you have
}
