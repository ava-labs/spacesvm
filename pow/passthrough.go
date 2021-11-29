// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package pow

import (
	"context"
)

var _ Checker = &passthrough{}

// NewPassthrough implements no-op checker.
func NewPassthrough() *passthrough {
	return &passthrough{}
}

type passthrough struct{}

func (p *passthrough) Check(_ context.Context, _ Unit) bool {
	return true
}
