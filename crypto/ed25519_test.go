// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package crypto

import (
	"testing"
)

func TestVerify(t *testing.T) {
	pk, err := NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	sig, err := pk.Sign([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if !Verify(pk.PublicKey().Bytes(), []byte("hello"), sig) {
		t.Fatal("failed to verify")
	}
}
