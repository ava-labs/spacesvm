// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"errors"
)

var (
	// Genesis Correctness
	ErrMissingGenesis            = errors.New("genesis is missing")
	ErrInvalidGenesisParent      = errors.New("genesis block parent is incorrect")
	ErrInvalidGenesisHeight      = errors.New("genesis block height is incorrect")
	ErrInvalidGenesisTimestamp   = errors.New("genesis block timestamp is incorrect")
	ErrInvalidGenesisDifficulty  = errors.New("genesis block difficulty is incorrect")
	ErrInvalidGenesisCost        = errors.New("genesis block cost is incorrect")
	ErrInvalidGenesisTxs         = errors.New("genesis block txs is incorrect")
	ErrInvalidGenesisBeneficiary = errors.New("genesis block beneficiary is incorrect")

	// Block Correctness
	ErrInvalidGenesis         = errors.New("invalid genesis")
	ErrTimestampTooEarly      = errors.New("block timestamp too early")
	ErrTimestampTooLate       = errors.New("block timestamp too late")
	ErrNoTxs                  = errors.New("no transactions")
	ErrInvalidCost            = errors.New("invalid block cost")
	ErrInsufficientSurplus    = errors.New("insufficient surplus difficulty")
	ErrParentBlockNotVerified = errors.New("parent block not verified or accepted")

	// Tx Correctness
	ErrInvalidSender     = errors.New("invalid sender")
	ErrInvalidBlockID    = errors.New("invalid blockID")
	ErrInvalidDifficulty = errors.New("invalid difficulty")
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrDuplicateTx       = errors.New("duplicate transaction")

	// Execution Correctness
	ErrValueTooBig       = errors.New("value too big")
	ErrPrefixExpired     = errors.New("prefix expired")
	ErrKeyMissing        = errors.New("key missing")
	ErrInvalidKey        = errors.New("key is invalid")
	ErrPublicKeyMismatch = errors.New("public key does not match decoded prefix")
	ErrPrefixNotExpired  = errors.New("prefix not expired")
	ErrPrefixMissing     = errors.New("prefix missing")
	ErrUnauthorized      = errors.New("sender is not authorized")

	// Crypto
	ErrInvalidPKLen = errors.New("invalid public key length")
)
