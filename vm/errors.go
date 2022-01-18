// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
)

var (
	ErrNoPendingTx    = errors.New("no pending tx")
	ErrTypedDataIsNil = errors.New("typed data is nil")
	ErrInputIsNil     = errors.New("input is nil")
	ErrInvalidEmptyTx = errors.New("invalid empty transaction")
	ErrCorruption     = errors.New("corruption detected")
)
