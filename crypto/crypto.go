// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package crypto defines interfaces for cryptographic mechanisms.
package crypto

// TODO: support other types of keys

type PublicKey interface {
	Verify(message, signature []byte) bool
	VerifyHash(hash, signature []byte) bool

	Address() string
	Bytes() []byte
}

type PrivateKey interface {
	PublicKey() PublicKey

	Sign(message []byte) ([]byte, error)
	SignHash(hash []byte) ([]byte, error)

	Bytes() []byte
}
