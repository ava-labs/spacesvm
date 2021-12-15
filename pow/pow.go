// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package pow implements Proof-of-Work primitives.
package pow

import (
	"math/big"

	"github.com/ava-labs/avalanchego/utils/hashing"
)

const (
	maxDifficulty = 256
)

// TODO: ASIC-Resistant?
// BenchmarkFinalHash
// BenchmarkFinalHash/SHA256
// BenchmarkFinalHash/SHA256-16         	 1711840	       675.1 ns/op
// BenchmarkFinalHash/Cryptonight
// BenchmarkFinalHash/Cryptonight-16    	      61	  20023176 ns/op
// BenchmarkFinalHash/BLAKE-256
// BenchmarkFinalHash/BLAKE-256-16      	 1237605	       943.4 ns/op
// BenchmarkFinalHash/Grøstl-256
// BenchmarkFinalHash/Grøstl-256-16     	  128697	      9012 ns/op
// BenchmarkFinalHash/JH-256
// BenchmarkFinalHash/JH-256-16         	  180705	      6561 ns/op
// BenchmarkFinalHash/Skein-256
// BenchmarkFinalHash/Skein-256-16      	  852234	      1478 ns/op

// TODO: make this more complicated
func Difficulty(b []byte) uint64 {
	h := hashing.ComputeHash256(b)
	n := new(big.Int).SetBytes(h)
	return uint64(maxDifficulty - n.BitLen())
}
