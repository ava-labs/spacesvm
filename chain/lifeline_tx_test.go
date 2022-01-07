// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"errors"
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
)

func TestLifelineTx(t *testing.T) {
	t.Parallel()

	priv, err := f.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	sender, err := FormatPK(priv.PublicKey())
	if err != nil {
		t.Fatal(err)
	}

	priv2, err := crypto.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pub2 := priv2.PublicKey()

	db := memdb.New()
	defer db.Close()

	tt := []struct {
		utx       UnsignedTransaction
		blockTime int64
		err       error
	}{
		{ // invalid when prefix info is missing
			utx:       &LifelineTx{BaseTx: &BaseTx{Sender: sender, Prefix: []byte("foo")}},
			blockTime: 1,
			err:       ErrPrefixMissing,
		},
		{ // successful claim with expiry time "blockTime" + "expiryTime"
			utx:       &ClaimTx{BaseTx: &BaseTx{Sender: sender, Prefix: []byte("foo")}},
			blockTime: 1,
			err:       nil,
		},
		{ // successful lifeline when prefix info is not missing
			utx:       &LifelineTx{BaseTx: &BaseTx{Sender: sender, Prefix: []byte("foo")}},
			blockTime: 1,
			err:       nil,
		},
		{ // invalid when lifelined by a different owner
			utx:       &LifelineTx{BaseTx: &BaseTx{Sender: pub2.Bytes(), Prefix: []byte("foo")}},
			blockTime: 1,
			err:       ErrUnauthorized,
		},
	}
	for i, tv := range tt {
		err := tv.utx.Execute(db, tv.blockTime)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.err, err)
		}
	}
}
