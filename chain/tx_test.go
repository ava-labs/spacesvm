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
			MinedTransaction: &MinedTransaction{
				UnsignedTransaction: &ClaimTx{
					BaseTx: &BaseTx{
						Prefix: bytes.Repeat([]byte{'b'}, i*10),
					},
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

func TestTransactionErrInvalidSignature(t *testing.T) {
	t.Parallel()

	tt := []struct {
		createTx   func() *Transaction
		blockTime  int64
		ctx        *Context
		executeErr error
	}{
		{
			createTx: func() *Transaction {
				return createTestTx(t, ids.ID{0, 1})
			},
			blockTime:  1,
			ctx:        &Context{RecentBlockIDs: ids.Set{{0, 1}: struct{}{}}},
			executeErr: nil,
		},
		{
			createTx: func() *Transaction {
				tx := createTestTx(t, ids.ID{0, 1})
				tx.Signature = []byte("invalid")
				return tx
			},
			blockTime:  1,
			ctx:        &Context{RecentBlockIDs: ids.Set{{0, 1}: struct{}{}}},
			executeErr: ErrInvalidSignature,
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

func createTestTx(t *testing.T, blockID ids.ID) *Transaction {
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
		MinedTransaction: &MinedTransaction{
			UnsignedTransaction: &ClaimTx{
				BaseTx: &BaseTx{
					Sender:  sender,
					Prefix:  []byte{'a'},
					BlockID: blockID,
				},
			},
			Graffiti: []uint64{0},
		},
	}
	if err := tx.Init(); err != nil {
		t.Fatal(err)
	}

	tx.Signature, err = priv.Sign(tx.MinedBytes())
	if err != nil {
		t.Fatal(err)
	}

	return tx
}
