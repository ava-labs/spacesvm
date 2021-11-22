// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package ed25519 implements cryptography utilities
// with Edwards-curve Digital Signature Algorithm (EdDSA).
package ed25519

import (
	"crypto/ed25519"
	"errors"

	"github.com/ava-labs/avalanchego/utils/formatting"
)

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

// NewPrivateKey implements the Factory interface
func NewPrivateKey() (PrivateKey, error) {
	_, k, err := ed25519.GenerateKey(nil)
	return &PrivateKeyED25519{sk: k}, err
}

// LoadPrivateKey loads a private key
func LoadPrivateKey(k []byte) (PrivateKey, error) {
	if len(k) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key size")
	}
	return &PrivateKeyED25519{sk: k}, nil
}

type PublicKeyED25519 struct {
	pk   ed25519.PublicKey
	addr string
}

// Verify implements the PublicKey interface
func (k *PublicKeyED25519) Verify(msg, sig []byte) bool {
	return ed25519.Verify(k.pk, msg, sig)
}

// VerifyHash implements the PublicKey interface
func (k *PublicKeyED25519) VerifyHash(hash, sig []byte) bool {
	return k.Verify(hash, sig)
}

// Address implements the PublicKey interface
func (k *PublicKeyED25519) Address() string {
	if len(k.addr) == 0 {
		addr, err := formatting.EncodeWithChecksum(formatting.CB58, k.pk)
		if err != nil {
			panic(err)
		}
		k.addr = addr
	}
	return k.addr
}

// Bytes implements the PublicKey interface
func (k *PublicKeyED25519) Bytes() []byte { return k.pk }

type PrivateKeyED25519 struct {
	sk ed25519.PrivateKey
	pk *PublicKeyED25519
}

// PublicKey implements the PrivateKey interface
func (k *PrivateKeyED25519) PublicKey() PublicKey {
	if k.pk == nil {
		k.pk = &PublicKeyED25519{
			pk: k.sk.Public().(ed25519.PublicKey),
		}
	}
	return k.pk
}

// Sign implements the PrivateKey interface
func (k *PrivateKeyED25519) Sign(msg []byte) ([]byte, error) {
	return ed25519.Sign(k.sk, msg), nil
}

// SignHash implements the PrivateKey interface
func (k PrivateKeyED25519) SignHash(hash []byte) ([]byte, error) {
	return k.Sign(hash)
}

// Bytes implements the PrivateKey interface
func (k PrivateKeyED25519) Bytes() []byte { return k.sk }
