// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/quarkvm/crypto"
)

func TestClaimTx(t *testing.T) {
	t.Parallel()

	priv, err := crypto.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.PublicKey()

	db := memdb.New()
	tt := []struct {
		tx        *ClaimTx
		blockTime int64
		err       error
	}{
		{ // invalid claim, [32]byte prefix is reserved for pubkey
			tx:        &ClaimTx{BaseTx: &BaseTx{Sender: pub.Bytes(), Prefix: bytes.Repeat([]byte{'a'}, crypto.PublicKeySize)}},
			blockTime: 1,
			err:       ErrPublicKeyMismatch,
		},
		{ // successful claim with expiry time "blockTime" + "expiryTime"
			tx:        &ClaimTx{BaseTx: &BaseTx{Sender: pub.Bytes(), Prefix: []byte("foo")}},
			blockTime: 1,
			err:       nil,
		},
		{ // invalid claim due to expiration
			tx:        &ClaimTx{BaseTx: &BaseTx{Sender: pub.Bytes(), Prefix: []byte("foo")}},
			blockTime: 1,
			err:       ErrPrefixNotExpired,
		},
		{ // successful new claim
			tx:        &ClaimTx{BaseTx: &BaseTx{Sender: pub.Bytes(), Prefix: []byte("foo")}},
			blockTime: 100,
			err:       nil,
		},
	}
	for i, tv := range tt {
		err := tv.tx.Execute(db, tv.blockTime)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.err, err)
		}
	}
}
