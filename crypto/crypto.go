// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package crypto

import (
	"bytes"
	"crypto/ed25519"
)

const (
	PublicKeySize = ed25519.PublicKeySize
)

var emptyPublicKeyBytes [PublicKeySize]byte

func IsEmptyPublicKey(pub []byte) bool {
	return bytes.Equal(pub, emptyPublicKeyBytes[:])
}

type PublicKey struct {
	PublicKey ed25519.PublicKey `serialize:"true" json:"publicKey"`
	Addr      string            `serialize:"true" json:"addr"`
}

type PrivateKey struct {
	PrivateKey ed25519.PrivateKey `serialize:"true" json:"privateKey"`

	pk *PublicKey
}
