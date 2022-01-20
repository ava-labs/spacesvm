// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
)

const (
	vOffset      = 64
	legacySigAdj = 27
)

func Sign(dh []byte, priv *ecdsa.PrivateKey) ([]byte, error) {
	sig, err := crypto.Sign(dh, priv)
	if err != nil {
		return nil, err
	}
	sig[vOffset] += legacySigAdj
	return sig, nil
}

func DeriveSender(dh []byte, sig []byte) (*ecdsa.PublicKey, error) {
	if len(sig) != crypto.SignatureLength {
		return nil, ErrInvalidSignature
	}
	// Avoid modifying the signature in place in case it is used elsewhere
	sigcpy := make([]byte, crypto.SignatureLength)
	copy(sigcpy, sig)

	// Support signers that don't apply offset (ex: ledger)
	if sigcpy[vOffset] >= legacySigAdj {
		sigcpy[vOffset] -= legacySigAdj
	}
	return crypto.SigToPub(dh, sigcpy)
}
