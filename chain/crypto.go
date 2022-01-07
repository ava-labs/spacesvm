// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import "github.com/ava-labs/avalanchego/utils/crypto"

var f *crypto.FactorySECP256K1R

func init() {
	f = &crypto.FactorySECP256K1R{}
}

func FormatPK(pk crypto.PublicKey) ([crypto.SECP256K1RPKLen]byte, error) {
	b := [crypto.SECP256K1RPKLen]byte{}
	pkb := pk.Bytes()
	if len(pkb) != crypto.SECP256K1RPKLen {
		return b, ErrInvalidPKLen
	}
	copy(b[:], pkb)
	return b, nil
}
