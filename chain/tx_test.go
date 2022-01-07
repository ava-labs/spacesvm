// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/ids"
)

func TestTransaction(t *testing.T) {
	t.Parallel()

	found := ids.NewSet(3)
	for i := range []int{0, 1, 2} {
		tx := &Transaction{
			UnsignedTransaction: &ClaimTx{
				BaseTx: &BaseTx{
					Prefix: bytes.Repeat([]byte{'b'}, i*10),
				},
			},
		}
		if err := tx.Init(); err != nil {
			t.Fatal(err)
		}
		if found.Contains(tx.ID()) {
			t.Fatal("duplicate transaction ID")
		}
		found.Add(tx.ID())
	}
}

func TestTransactionExecute(t *testing.T) {
	t.Parallel()

	tt := []struct {
		createTx   func() *Transaction
		blockTime  int64
		ctx        *Context
		executeErr error
	}{
		{
			createTx: func() *Transaction {
				return createTestClaimTx(t, ids.ID{0, 1}, 1000)
			},
			blockTime: 1,
			ctx: &Context{
				RecentBlockIDs: ids.Set{{0, 1}: struct{}{}},
				MinExpiry:      60,
			},
			executeErr: nil,
		},
		{
			createTx: func() *Transaction {
				tx := createTestClaimTx(t, ids.ID{0, 1}, 100)
				tx.Signature = []byte("invalid")
				return tx
			},
			blockTime: 1,
			ctx: &Context{
				RecentBlockIDs: ids.Set{{0, 1}: struct{}{}},
				MinExpiry:      60,
			},
			executeErr: ErrInvalidSignature,
		},
		{
			createTx: func() *Transaction {
				return createTestClaimTx(t, ids.ID{0, 1}, 500)
			},
			blockTime: 1,
			ctx: &Context{
				RecentBlockIDs: ids.Set{{0, 1}: struct{}{}},
				MinExpiry:      1000,
			},
			executeErr: ErrInvalidExpiry,
		},
	}
	for i, tv := range tt {
		tx := tv.createTx()
		err := tx.Execute(memdb.New(), tv.blockTime, tv.ctx)
		if !errors.Is(err, tv.executeErr) {
			t.Fatalf("#%d: unexpected tx.Execute error %v, expected %v", i, err, tv.executeErr)
		}
	}
}

func createTestClaimTx(t *testing.T, blockID ids.ID, expiry uint64) *Transaction {
	t.Helper()

	priv, err := f.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	sender, err := FormatPK(priv.PublicKey())
	if err != nil {
		t.Fatal(err)
	}

	tx := &Transaction{
		UnsignedTransaction: &ClaimTx{
			BaseTx: &BaseTx{
				Sender:  sender,
				Prefix:  []byte{'a'},
				BlockID: blockID,
			},
			Expiry: expiry,
		},
	}
	if err := tx.Init(); err != nil {
		t.Fatal(err)
	}

	tx.Signature, err = priv.Sign(tx.UnsignedBytes())
	if err != nil {
		t.Fatal(err)
	}

	return tx
}
