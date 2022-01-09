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
	txm := mempool.New(4)
	for _, i := range []int{1, 2, 3} { // difficulty 2, 3, 0
		tx := &chain.Transaction{
			Signature: bytes.Repeat([]byte{'a'}, i*10),
			UnsignedTransaction: &chain.SetTx{
				BaseTx: &chain.BaseTx{
					Prefix:   bytes.Repeat([]byte{'b'}, i*10),
					Graffiti: 4,
				},
			},
		}
		if err := tx.Init(); err != nil {
			t.Fatal(err)
		}
		if !txm.Add(tx) {
			t.Fatalf("tx %s was not added", tx.ID())
		}
	}
	if _, diff := txm.PeekMax(); diff != 1 {
		t.Fatalf("difficulty expected 1, got %d", diff)
	}
	if _, diff := txm.PeekMin(); diff != 0 {
		t.Fatalf("difficulty expected 0, got %d", diff)
	}
}
