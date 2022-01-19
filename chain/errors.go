// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"errors"
)

var (
	// Genesis Correctness
	ErrInvalidMagic     = errors.New("invalid magic")
	ErrInvalidBlockRate = errors.New("invalid block rate")

	// Block Correctness
	ErrTimestampTooEarly      = errors.New("block timestamp too early")
	ErrTimestampTooLate       = errors.New("block timestamp too late")
	ErrNoTxs                  = errors.New("no transactions")
	ErrInvalidCost            = errors.New("invalid block cost")
	ErrInvalidPrice           = errors.New("invalid price")
	ErrInsufficientSurplus    = errors.New("insufficient surplus fee")
	ErrParentBlockNotVerified = errors.New("parent block not verified or accepted")

	// Tx Correctness
	ErrInvalidBlockID      = errors.New("invalid blockID")
	ErrInvalidSignature    = errors.New("invalid signature")
	ErrDuplicateTx         = errors.New("duplicate transaction")
	ErrInsufficientPrice   = errors.New("insufficient price")
	ErrInvalidType         = errors.New("invalid tx type")
	ErrTypedDataKeyMissing = errors.New("typed data key missing")

	// Execution Correctness
	ErrValueEmpty      = errors.New("value empty")
	ErrValueTooBig     = errors.New("value too big")
	ErrSpaceExpired    = errors.New("space expired")
	ErrKeyMissing      = errors.New("key missing")
	ErrInvalidKey      = errors.New("key is invalid")
	ErrAddressMismatch = errors.New("address does not match decoded space")
	ErrSpaceNotExpired = errors.New("space not expired")
	ErrSpaceMissing    = errors.New("space missing")
	ErrUnauthorized    = errors.New("sender is not authorized")
	ErrInvalidBalance  = errors.New("invalid balance")
	ErrNonActionable   = errors.New("transaction doesn't do anything")
	ErrBlockTooBig     = errors.New("block too big")
)
