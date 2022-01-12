// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mempool_test

import (
	"bytes"
	"testing"

	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/mempool"
)

func TestMempool(t *testing.T) {
	g := chain.DefaultGenesis()
	txm := mempool.New(g, 4)
	for _, i := range []int{200, 220, 250} {
		tx := &chain.Transaction{
			Signature: bytes.Repeat([]byte{'a'}, i),
			UnsignedTransaction: &chain.SetTx{
				BaseTx: &chain.BaseTx{
					Prefix:   bytes.Repeat([]byte{'b'}, i),
					Graffiti: 28829,
				},
			},
		}
		if err := tx.Init(g); err != nil {
			t.Fatal(err)
		}
		if !txm.Add(tx) {
			t.Fatalf("tx %s was not added", tx.ID())
		}
	}
	if _, diff := txm.PeekMax(); diff != 3 {
		t.Fatalf("difficulty expected 3, got %d", diff)
	}
	if _, diff := txm.PeekMin(); diff != 0 {
		t.Fatalf("difficulty expected 0, got %d", diff)
	}
}
