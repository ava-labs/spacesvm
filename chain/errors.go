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
	ErrInvalidExtraData       = errors.New("invalid block extra data")
	ErrInsufficientSurplus    = errors.New("insufficient surplus difficulty")
	ErrParentBlockNotVerified = errors.New("parent block not verified or accepted")

	// Tx Correctness
	ErrInvalidSender     = errors.New("invalid sender")
	ErrInvalidBlockID    = errors.New("invalid blockID")
	ErrInvalidDifficulty = errors.New("invalid difficulty")
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrInvalidExpiry     = errors.New("invalid expiry")
	ErrDuplicateTx       = errors.New("duplicate transaction")

	// Execution Correctness
	ErrValueTooBig       = errors.New("value too big")
	ErrPrefixExpired     = errors.New("prefix expired")
	ErrKeyMissing        = errors.New("key missing")
	ErrPublicKeyMismatch = errors.New("public key does not match decoded prefix")
	ErrPrefixNotExpired  = errors.New("prefix not expired")
	ErrPrefixMissing     = errors.New("prefix missing")
	ErrUnauthorized      = errors.New("sender is not authorized")

	// Crypto
	ErrInvalidPKLen = errors.New("invalid public key length")
)
