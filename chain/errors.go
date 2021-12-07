package chain

import (
	"errors"
)

var (
	ErrPrefixEmpty            = errors.New("prefix cannot be empty")
	ErrPrefixTooBig           = errors.New("prefix too big")
	ErrPrefixContainsDelim    = errors.New("prefix contains delimiter")
	ErrInvalidSender          = errors.New("invalid sender")
	ErrInvalidBlockID         = errors.New("invalid blockID")
	ErrPrefixMissing          = errors.New("prefix missing")
	ErrKeyEmpty               = errors.New("key cannot be empty")
	ErrKeyTooBig              = errors.New("key too big")
	ErrValueTooBig            = errors.New("value too big")
	ErrUnauthorized           = errors.New("sender is not authorized")
	ErrPrefixExpired          = errors.New("prefix expired")
	ErrKeyMissing             = errors.New("key missing")
	ErrDuplicateTx            = errors.New("duplicate transaction")
	ErrInvalidDifficulty      = errors.New("invalid difficulty")
	ErrInvalidSignature       = errors.New("invalid signature")
	ErrTimestampTooEarly      = errors.New("block timestamp too early")
	ErrTimestampTooLate       = errors.New("block timestamp too late")
	ErrNoTxs                  = errors.New("no transactions")
	ErrInvalidCost            = errors.New("invalid block cost")
	ErrInsufficientSurplus    = errors.New("insufficient surplus difficulty")
	ErrParentBlockNotVerified = errors.New("parent block not verified or accepted")
	ErrPublicKeyMismatch      = errors.New("public key does not match decoded prefix")
	ErrPrefixNotExpired       = errors.New("prefix not expired")
)
