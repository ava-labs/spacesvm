// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
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
				BaseTx: &BaseTx{},
				Space:  strings.Repeat("b", i*10),
			},
		}
		fmt.Println(tx.UnsignedTransaction.TypedData())
		dh, err := DigestHash(tx.UnsignedTransaction)
		if err != nil {
			t.Fatal(err)
		}
		if len(dh) != 32 {
			t.Fatalf("hash insufficient d=%d", len(dh))
		}
		tx.Signature, err = Sign(dh, priv)
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

func TestTransactionErrInvalidSignature(t *testing.T) {
	t.Parallel()

	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	sender := crypto.PubkeyToAddress(priv.PublicKey)

	priv2, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	g := DefaultGenesis()
	tt := []struct {
		createTx   func() *Transaction
		blockTime  int64
		ctx        *Context
		executeErr error
	}{
		{
			createTx: func() *Transaction {
				return createTestTx(t, ids.ID{0, 1}, priv)
			},
			blockTime:  1,
			ctx:        &Context{RecentBlockIDs: ids.Set{{0, 1}: struct{}{}}},
			executeErr: nil,
		},
		{
			createTx: func() *Transaction {
				tx := createTestTx(t, ids.ID{0, 1}, priv2)
				return tx
			},
			blockTime:  1,
			ctx:        &Context{RecentBlockIDs: ids.Set{{0, 1}: struct{}{}}},
			executeErr: ErrInvalidBalance,
		},
	}
	for i, tv := range tt {
		db := memdb.New()
		g.CustomAllocation = []*CustomAllocation{
			{
				Address: sender,
				Balance: 10000000,
			},
			// sender2 is not given any balance
		}
		if err := g.Load(db, nil); err != nil {
			t.Fatal(err)
		}
		tx := tv.createTx()
		dummy := DummyBlock(tv.blockTime, tx)
		err := tx.Execute(g, db, dummy, tv.ctx)
		if !errors.Is(err, tv.executeErr) {
			t.Fatalf("#%d: unexpected tx.Execute error %v, expected %v", i, err, tv.executeErr)
		}
	}
}

func createTestTx(t *testing.T, blockID ids.ID, priv *ecdsa.PrivateKey) *Transaction {
	t.Helper()

	tx := &Transaction{
		UnsignedTransaction: &ClaimTx{
			BaseTx: &BaseTx{
				BlockID: blockID,
				Price:   10,
			},
			Space: "a",
		},
	}
	dh, err := DigestHash(tx.UnsignedTransaction)
	if err != nil {
		t.Fatal(err)
	}
	if len(dh) != 32 {
		t.Fatalf("hash insufficient d=%d", len(dh))
	}
	sig, err := Sign(dh, priv)
	if err != nil {
		t.Fatal(err)
	}
	tx.Signature = sig
	if err := tx.Init(DefaultGenesis()); err != nil {
		t.Fatal(err)
	}

	return tx
}
