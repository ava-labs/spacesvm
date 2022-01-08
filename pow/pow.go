// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package pow implements Proof-of-Work primitives.
package pow

import (
	"math/big"

	"golang.org/x/crypto/sha3"
)

var (
	MaxDifficulty, _ = new(big.Int).SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
)

func Difficulty(b []byte) *big.Int {
	h := sha3.Sum256(b)
	v := new(big.Int).SetBytes(h[:])
	return new(big.Int).Sub(MaxDifficulty, v)
}
