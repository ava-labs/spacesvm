// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package pow

import (
	"context"
	"testing"
	"time"
)

func TestChecker(t *testing.T) {
	cc := New(func() uint64 { return 20 })

	// requires only 10 nonces, so 1-min should be enough
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if !cc.Check(ctx, NewUnit([]byte("hello"))) {
		t.Fatal("failed to check")
	}
}
