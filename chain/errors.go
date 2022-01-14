// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"errors"
)

var (
	// Block Correctness
	ErrTimestampTooEarly      = errors.New("block timestamp too early")
	ErrTimestampTooLate       = errors.New("block timestamp too late")
	ErrNoTxs                  = errors.New("no transactions")
	ErrInvalidCost            = errors.New("invalid block cost")
	ErrInvalidPrice           = errors.New("invalid price")
	ErrInsufficientSurplus    = errors.New("insufficient surplus difficulty")
	ErrParentBlockNotVerified = errors.New("parent block not verified or accepted")

	// Tx Correctness
	ErrInvalidMagic      = errors.New("invalid magic")
	ErrInvalidBlockID    = errors.New("invalid blockID")
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrDuplicateTx       = errors.New("duplicate transaction")
	ErrInsufficientPrice = errors.New("insufficient price")

	// Execution Correctness
	ErrValueTooBig      = errors.New("value too big")
	ErrPrefixExpired    = errors.New("prefix expired")
	ErrKeyMissing       = errors.New("key missing")
	ErrInvalidKey       = errors.New("key is invalid")
	ErrAddressMismatch  = errors.New("address does not match decoded prefix")
	ErrPrefixNotExpired = errors.New("prefix not expired")
	ErrPrefixMissing    = errors.New("prefix missing")
	ErrUnauthorized     = errors.New("sender is not authorized")
	ErrInvalidBalance   = errors.New("invalid balance")
	ErrNonActionable    = errors.New("transaction doesn't do anything")
)
