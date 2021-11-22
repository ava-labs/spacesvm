// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package ed25519 implements cryptography utilities
// with Edwards-curve Digital Signature Algorithm (EdDSA).
package ed25519

import (
	"crypto/ed25519"
	"errors"

	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/quarkvm/crypto"
	log "github.com/inconshreveable/log15"
)

var (
	_ crypto.PrivateKey = &PrivateKey{}
	_ crypto.PublicKey  = &PublicKey{}
)

// NewPrivateKey implements the Factory interface
func NewPrivateKey() (crypto.PrivateKey, error) {
	_, k, err := ed25519.GenerateKey(nil)
	return &PrivateKey{sk: k}, err
}

// LoadPrivateKey loads a private key
func LoadPrivateKey(k []byte) (crypto.PrivateKey, error) {
	if len(k) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key size")
	}
	return &PrivateKey{sk: k}, nil
}

type PublicKey struct {
	PublicKey ed25519.PublicKey `serialize:"true" json:"publicKey"`
	Addr      string            `serialize:"true" json:"addr"`
}

// Verify implements the PublicKey interface
func (k *PublicKey) Verify(msg, sig []byte) bool {
	return ed25519.Verify(k.PublicKey, msg, sig)
}

// VerifyHash implements the PublicKey interface
func (k *PublicKey) VerifyHash(hash, sig []byte) bool {
	return k.Verify(hash, sig)
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
func (k *PublicKey) Bytes() []byte { return k.PublicKey }

type PrivateKey struct {
	sk ed25519.PrivateKey
	pk *PublicKey
}

// PublicKey implements the PrivateKey interface
func (k *PrivateKey) PublicKey() crypto.PublicKey {
	if k.pk == nil {
		k.pk = &PublicKey{
			PublicKey: k.sk.Public().(ed25519.PublicKey),
		}
	}
	return k.pk
}

// Sign implements the PrivateKey interface
func (k *PrivateKey) Sign(msg []byte) ([]byte, error) {
	return ed25519.Sign(k.sk, msg), nil
}

// SignHash implements the PrivateKey interface
func (k PrivateKey) SignHash(hash []byte) ([]byte, error) {
	return k.Sign(hash)
}

// Bytes implements the PrivateKey interface
func (k PrivateKey) Bytes() []byte { return k.sk }

func Verify(pub []byte, msg []byte, sig []byte) bool {
	log.Debug("ed25519.Verify", "pub", len(pub))
	return ed25519.Verify(ed25519.PublicKey(pub), msg, sig)
}
