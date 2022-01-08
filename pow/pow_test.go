// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package pow

import (
	"encoding/binary"
	"testing"
)

func benchmarkDifficulty(d uint64, b *testing.B) {
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

func BenchmarkDifficulty1(b *testing.B)    { benchmarkDifficulty(1, b) }
func BenchmarkDifficulty10(b *testing.B)   { benchmarkDifficulty(10, b) }
func BenchmarkDifficulty50(b *testing.B)   { benchmarkDifficulty(50, b) }
func BenchmarkDifficulty100(b *testing.B)  { benchmarkDifficulty(100, b) }
func BenchmarkDifficulty500(b *testing.B)  { benchmarkDifficulty(500, b) }
func BenchmarkDifficulty1000(b *testing.B) { benchmarkDifficulty(1000, b) }
