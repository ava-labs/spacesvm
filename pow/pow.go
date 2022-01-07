// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package pow implements Proof-of-Work primitives.
package pow

import (
	"math/big"

	"golang.org/x/crypto/sha3"
)

const (
	maxDifficulty = 256
)

func Difficulty(b []byte) uint64 {
	h := sha3.Sum256(b)
	n := new(big.Int).SetBytes(h[:])
	return uint64(maxDifficulty - n.BitLen())
}
