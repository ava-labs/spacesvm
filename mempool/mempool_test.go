// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mempool_test

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/mempool"
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
					Price: uint64(i),
				},
				Space: strings.Repeat("a", i),
			},
		}
		dh, err := chain.DigestHash(tx.UnsignedTransaction)
		if err != nil {
			t.Fatal(err)
		}
		sig, err := chain.Sign(dh, priv)
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
	if _, price := txm.PeekMax(); price != 250 {
		t.Fatalf("price expected 250, got %d", price)
	}
	if _, price := txm.PeekMin(); price != 200 {
		t.Fatalf("price expected 200, got %d", price)
	}
	if length := txm.Len(); length != 3 {
		t.Fatalf("length expected 3, got %d", length)
	}
}
