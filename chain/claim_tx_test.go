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

	priv2, err := crypto.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pub2 := priv2.PublicKey()

	db := memdb.New()
	defer db.Close()

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
		{ // successful new claim by different owner
			tx:        &ClaimTx{BaseTx: &BaseTx{Sender: pub2.Bytes(), Prefix: []byte("foo")}},
			blockTime: 150,
			err:       nil,
		},
		{ // invalid claim due to expiration by different owner
			tx:        &ClaimTx{BaseTx: &BaseTx{Sender: pub2.Bytes(), Prefix: []byte("foo")}},
			blockTime: 177,
			err:       ErrPrefixNotExpired,
		},
	}
	for i, tv := range tt {
		if i > 0 {
			// Expire old prefixes between txs
			if err := ExpireNext(db, tt[i-1].blockTime, tv.blockTime); err != nil {
				t.Fatalf("#%d: ExpireNext errored %v", i, err)
			}
		}
		err := tv.tx.Execute(db, tv.blockTime)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.err, err)
		}
		if tv.err != nil {
			continue
		}
		info, exists, err := GetPrefixInfo(db, tv.tx.Prefix)
		if err != nil {
			t.Fatalf("#%d: failed to get prefix info %v", i, err)
		}
		if !exists {
			t.Fatalf("#%d: failed to find prefix info", i)
		}
		if !bytes.Equal(info.Owner[:], tv.tx.Sender[:]) {
			t.Fatalf("#%d: unexpected owner found (expected pub key %q)", i, string(pub.PublicKey))
		}
	}
}
