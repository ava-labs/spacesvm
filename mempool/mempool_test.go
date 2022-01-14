// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mempool_test

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/mempool"
)

func TestMempool(t *testing.T) {
	g := chain.DefaultGenesis()
	txm := mempool.New(g, 3)
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range []int{100, 200, 220, 250} {
		tx := &chain.Transaction{
			UnsignedTransaction: &chain.SetTx{
				BaseTx: &chain.BaseTx{
					Pfx:  bytes.Repeat([]byte{'b'}, i),
					Prce: uint64(i),
				},
			},
		}
		sig, err := crypto.Sign(tx.DigestHash(), priv)
		if err != nil {
			t.Fatal(err)
		}
		tx.Signature = sig
		if err := tx.Init(g); err != nil {
			t.Fatal(err)
		}
		if !txm.Add(tx) {
			t.Fatalf("tx %s was not added", tx.ID())
		}
	}
	if _, diff := txm.PeekMax(); diff != 250 {
		t.Fatalf("difficulty expected 250, got %d", diff)
	}
	if _, diff := txm.PeekMin(); diff != 200 {
		t.Fatalf("difficulty expected 200, got %d", diff)
	}
	if length := txm.Len(); length != 3 {
		t.Fatalf("length expected 3, got %d", length)
	}
}
