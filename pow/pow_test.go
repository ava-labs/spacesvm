// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package pow

import (
	"encoding/binary"
	"testing"
)

func benchmarkDifficulty(b *testing.B, d uint64) {
	for n := 0; n < b.N; n++ {
		for i := uint64(0); ; i++ {
			b := [16]byte{}
			binary.LittleEndian.PutUint64(b[:], uint64(n))
			binary.LittleEndian.PutUint64(b[8:], i)
			if Difficulty(b[:]) >= d {
				break
			}
		}
	}
}

func BenchmarkDifficulty1(b *testing.B)    { benchmarkDifficulty(b, 1) }
func BenchmarkDifficulty10(b *testing.B)   { benchmarkDifficulty(b, 10) }
func BenchmarkDifficulty50(b *testing.B)   { benchmarkDifficulty(b, 50) }
func BenchmarkDifficulty100(b *testing.B)  { benchmarkDifficulty(b, 100) }
func BenchmarkDifficulty500(b *testing.B)  { benchmarkDifficulty(b, 500) }
func BenchmarkDifficulty1000(b *testing.B) { benchmarkDifficulty(b, 1000) }
