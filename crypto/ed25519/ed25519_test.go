// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ed25519

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
	pub := pk.PublicKey()

	if !Verify(pub.Bytes(), []byte("hello"), sig) {
		t.Fatal("failed to verify")
	}
}
