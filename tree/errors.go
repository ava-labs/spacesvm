// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package tree

import (
	"errors"
)

var (
	ErrEmpty   = errors.New("file is empty")
	ErrMissing = errors.New("required file is missing")
)
