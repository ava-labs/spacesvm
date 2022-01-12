// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package pow implements Proof-of-Work primitives.
package pow

import (
	"math/big"

	"golang.org/x/crypto/sha3"
)

// Recommended reading to understand how this works: https://en.bitcoin.it/wiki/Difficulty
const (
	totalPrecision = 256
	// Bitcoin uses 2^208 for the [base]
	// Each unit of difficulty at 2^230 adds ~1ms on a new laptop
	difficultyBase = 230
	expectedBase   = totalPrecision - difficultyBase
)

var (
	big2           = big.NewInt(2)
	scalingOperand = big.NewInt(0xFFFF)

	diffFactor = new(big.Int).Mul(scalingOperand, new(big.Int).Exp(big2, big.NewInt(difficultyBase), nil))

	expectedFactor = new(big.Int).Exp(big2, big.NewInt(expectedBase), nil)
)

func Difficulty(b []byte) uint64 {
	h := sha3.Sum256(b)
	v := new(big.Int).SetBytes(h[:])
	r := new(big.Int).Div(diffFactor, v)
	return r.Uint64()
}

// ExpectedHashes provides an estimate of the number of hashes that must be
// computed for a given difficulty.
func ExpectedHashes(difficulty uint64) uint64 {
	n := new(big.Int).Mul(new(big.Int).SetUint64(difficulty), expectedFactor)
	r := new(big.Int).Div(n, scalingOperand)
	return r.Uint64()
}
