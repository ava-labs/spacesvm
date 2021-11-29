// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package pow

import (
	"testing"
)

func TestUnitCryptonight(t *testing.T) {
	u := NewUnit([]byte("hello"))
	tries := 0
	for u.Next() && !u.Prove(20) {
		if tries > 10 {
			t.Fatalf("expected to prove in 10 rounds (so far %d tries)", tries)
		}
		tries++
	}
}
