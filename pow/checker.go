// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package pow

import (
	"context"
)

type Checker interface {
	// Returns true if PoW is confirmed.
	// Blocks until mining via enumeration is done.
	// Set context timeout to fail fast in case it takes too long.
	Check(ctx context.Context, unit Unit) bool
}

func New(getDifficulty func() uint64) Checker {
	return &cryptonightChecker{getDifficulty: getDifficulty}
}

type cryptonightChecker struct {
	getDifficulty func() uint64
}

func (cc *cryptonightChecker) Check(ctx context.Context, unit Unit) (proved bool) {
	diff := cc.getDifficulty()
done:
	for unit.Next() {
		select {
		case <-ctx.Done():
			break done
		default:
		}
		if unit.Prove(diff) {
			proved = true
			break done
		}
	}
	return proved
}
