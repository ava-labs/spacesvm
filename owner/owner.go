// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package owner defines the key owner information.
package owner

import (
	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/crypto/ed25519"
)

func init() {
	codec.RegisterType(&Owner{})
}

type Owner struct {
	PublicKey ed25519.PublicKey `serialize:"true" json:"publicKey"`

	Namespace   string `serialize:"true" json:"namespace"`
	LastUpdated int64  `serialize:"true" json:"lastUpdated"`
	Expiry      int64  `serialize:"true" json:"expiry"`

	// decays faster the more keys you have
	Keys int64 `serialize:"true" json:"keys"`
}
