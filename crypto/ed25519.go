// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package ed25519 implements cryptography utilities
// with Edwards-curve Digital Signature Algorithm (EdDSA).
package crypto

import (
	"crypto/ed25519"
	"errors"

	"github.com/ava-labs/avalanchego/utils/formatting"
)

// NewPrivateKey implements the Factory interface
func NewPrivateKey() (*PrivateKey, error) {
	_, k, err := ed25519.GenerateKey(nil)
	return &PrivateKey{PrivateKey: k}, err
}

// LoadPrivateKey loads a private key
func LoadPrivateKey(k []byte) (*PrivateKey, error) {
	if len(k) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key size")
	}
	return &PrivateKey{PrivateKey: k}, nil
}

// Verify implements the PublicKey interface
func (k *PublicKey) Verify(msg, sig []byte) bool {
	return ed25519.Verify(k.PublicKey, msg, sig)
}

// Address implements the PublicKey interface
func (k *PublicKey) Address() string {
	if len(k.Addr) == 0 {
		addr, err := formatting.EncodeWithChecksum(formatting.CB58, k.PublicKey)
		if err != nil {
			panic(err)
		}
		k.Addr = addr
	}
	return k.Addr
}

// Bytes implements the PublicKey interface
func (k *PublicKey) Bytes() [PublicKeySize]byte {
	// TODO: probably a better way to do this
	var pk [PublicKeySize]byte
	copy(pk[:], k.PublicKey)
	return pk
}

// PublicKey implements the PrivateKey interface
func (k *PrivateKey) PublicKey() *PublicKey {
	if k.pk == nil {
		k.pk = &PublicKey{
			PublicKey: k.PrivateKey.Public().(ed25519.PublicKey),
		}
	}
	return k.pk
}

// Sign implements the PrivateKey interface
func (k *PrivateKey) Sign(msg []byte) ([]byte, error) {
	return ed25519.Sign(k.PrivateKey, msg), nil
}

// Bytes implements the PrivateKey interface
func (k PrivateKey) Bytes() []byte { return k.PrivateKey }

func Verify(pub [PublicKeySize]byte, msg []byte, sig []byte) bool {
	return ed25519.Verify(ed25519.PublicKey(pub[:]), msg, sig)
}
