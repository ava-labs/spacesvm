// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestTransaction(t *testing.T) {
	t.Parallel()

	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	found := ids.NewSet(3)
	g := DefaultGenesis()
	for i := range []int{0, 1, 2} {
		tx := &Transaction{
			UnsignedTransaction: &ClaimTx{
				BaseTx: &BaseTx{
					Pfx: bytes.Repeat([]byte{'b'}, i*10),
				},
			},
		}
		dh := tx.DigestHash()
		if len(tx.DigestHash()) != 32 {
			t.Fatal("hash insufficient")
		}
		tx.Signature, err = crypto.Sign(dh, priv)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Init(g); err != nil {
			t.Fatal(err)
		}
		if found.Contains(tx.ID()) {
			t.Fatal("duplicate transaction ID")
		}
		found.Add(tx.ID())
	}
}

// func TestTransactionErrInvalidSignature(t *testing.T) {
// 	t.Parallel()
//
// 	g := DefaultGenesis()
// 	tt := []struct {
// 		createTx   func() *Transaction
// 		blockTime  int64
// 		ctx        *Context
// 		executeErr error
// 	}{
// 		{
// 			createTx: func() *Transaction {
// 				return createTestTx(t, ids.ID{0, 1}, false)
// 			},
// 			blockTime:  1,
// 			ctx:        &Context{RecentBlockIDs: ids.Set{{0, 1}: struct{}{}}},
// 			executeErr: nil,
// 		},
// 		{
// 			createTx: func() *Transaction {
// 				tx := createTestTx(t, ids.ID{0, 1}, true)
// 				return tx
// 			},
// 			blockTime:  1,
// 			ctx:        &Context{RecentBlockIDs: ids.Set{{0, 1}: struct{}{}}},
// 			executeErr: ErrInvalidSignature,
// 		},
// 	}
// 	for i, tv := range tt {
// 		db := memdb.New()
// 		tx := tv.createTx()
// 		v, err := ModifyBalance(db, tx.Sender(), true, 100000)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		if v != 100000 {
// 			t.Fatal(ErrInvalidBalance)
// 		}
// 		err = tx.Execute(g, memdb.New(), tv.blockTime, tv.ctx)
// 		if !errors.Is(err, tv.executeErr) {
// 			t.Fatalf("#%d: unexpected tx.Execute error %v, expected %v", i, err, tv.executeErr)
// 		}
// 	}
// }

func createTestTx(t *testing.T, blockID ids.ID, invalid bool) *Transaction {
	t.Helper()

	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	tx := &Transaction{
		UnsignedTransaction: &ClaimTx{
			BaseTx: &BaseTx{
				Pfx:   []byte{'a'},
				BlkID: blockID,
			},
		},
	}
	if !invalid {
		dh := tx.DigestHash()
		if len(tx.DigestHash()) != 32 {
			t.Fatal("hash insufficient")
		}
		tx.Signature, err = crypto.Sign(dh, priv)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		tx.Signature = []byte("invalid")
	}

	if err := tx.Init(DefaultGenesis()); err != nil {
		t.Fatal(err)
	}

	return tx
}
