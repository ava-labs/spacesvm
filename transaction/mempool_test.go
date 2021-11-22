// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package transaction

import (
	"testing"
)

func TestMempool(t *testing.T) {
	tx1 := New(
		Unsigned{
			PublicKey: nil,
			Op:        "Put",
			Key:       "foo1",
			Value:     "hello world!",
		},
		[]byte("sig1"),
	) // difficulty 1

	tx2 := New(
		Unsigned{
			PublicKey: []byte("adsf213213213123123123123"),
			Op:        "Range",
			Key:       "foosdfasfsafs1",
		},
		[]byte("sig2"),
	) // difficulty 2

	tx3 := New(
		Unsigned{
			PublicKey: nil,
			Op:        "Put",
			Key:       "foo2",
			Value:     "dafadsf233414312312312312",
		},
		[]byte("sig3"),
	) // difficulty 3

	txm := NewMempool(3)
	txm.Push(tx1)
	txm.Push(tx2)
	txm.Push(tx3)

	_, diff := txm.PeekMax()
	if diff != 8 {
		t.Fatalf("difficulty expected 3, got %d", diff)
	}
	_, diff = txm.PeekMin()
	if diff != 1 {
		t.Fatalf("difficulty expected 1, got %d", diff)
	}
}
