// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package pow implements Proof-of-Work primitives.
package pow

import (
	"math/big"

	"golang.org/x/crypto/sha3"
)

// Recommended reading to understand how this works: https://en.bitcoin.it/wiki/Difficulty
var (
	// Bitcoin uses 2^208 for the [base]
	// Each unit of difficulty at 2^230 adds ~1ms on a new laptop
	base          = new(big.Int).Exp(big.NewInt(2), big.NewInt(230), nil)
	scalingFactor = new(big.Int).Mul(big.NewInt(0xFFFF), base)
)

func Difficulty(b []byte) uint64 {
	h := sha3.Sum256(b)
	v := new(big.Int).SetBytes(h[:])
	r := new(big.Int).Div(scalingFactor, v)
	return r.Uint64()
}

// ExpectedHashes provides an estimate of the number of hashes that must be
// computed for a given difficulty.
func ExpectedHashes(difficulty uint64) uint64 {
	// 256-240 = 16
	b := new(big.Int).Exp(big.NewInt(2), big.NewInt(16), nil)
	n := new(big.Int).Mul(new(big.Int).SetUint64(difficulty), b)
	r := new(big.Int).Div(n, big.NewInt(0xFFFF))
	return r.Uint64()
}
