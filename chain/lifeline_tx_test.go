// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"errors"
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/quarkvm/crypto"
)

func TestLifelineTx(t *testing.T) {
	t.Parallel()

	priv, err := crypto.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.PublicKey()

	db := memdb.New()
	defer db.Close()

	tt := []struct {
		utx       UnsignedTransaction
		blockTime int64
		err       error
	}{
		{ // invalid when prefix info is missing
			utx:       &LifelineTx{BaseTx: &BaseTx{Sender: pub.Bytes(), Prefix: []byte("foo")}},
			blockTime: 1,
			err:       ErrPrefixMissing,
		},
		{ // successful claim with expiry time "blockTime" + "expiryTime"
			utx:       &ClaimTx{BaseTx: &BaseTx{Sender: pub.Bytes(), Prefix: []byte("foo")}},
			blockTime: 1,
			err:       nil,
		},
		{ // successful lifeline when prefix info is missing
			utx:       &LifelineTx{BaseTx: &BaseTx{Sender: pub.Bytes(), Prefix: []byte("foo")}},
			blockTime: 1,
			err:       nil,
		},
	}
	for i, tv := range tt {
		err := tv.utx.Execute(db, tv.blockTime)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.err, err)
		}
	}
}
