// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package crypto_test

import (
	"testing"

	"github.com/ava-labs/quarkvm/crypto"
)

func TestVerify(t *testing.T) {
	t.Parallel()

	pk, err := crypto.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	sig, err := pk.Sign([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if !crypto.Verify(pk.PublicKey().Bytes(), []byte("hello"), sig) {
		t.Fatal("failed to verify")
	}
}
