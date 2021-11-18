// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package transaction

import (
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/crypto/ed25519"
	"github.com/ava-labs/quarkvm/storage"
)

func TestMempool(t *testing.T) {
	codec.RegisterType(&testUnsigned{})

	tx1 := newTestTransaction(ids.GenerateTestID(), 1, ids.GenerateTestID(), []byte("sig1")) // difficulty 1
	tx2 := newTestTransaction(ids.GenerateTestID(), 2, ids.GenerateTestID(), []byte("sig2")) // difficulty 2
	tx3 := newTestTransaction(ids.GenerateTestID(), 3, ids.GenerateTestID(), []byte("sig3")) // difficulty 3

	txm := NewMempool(3)
	txm.Push(tx1)
	txm.Push(tx2)
	txm.Push(tx3)

	_, diff := txm.PeekMax()
	if diff != 3 {
		t.Fatalf("difficulty expected 3, got %d", diff)
	}
	_, diff = txm.PeekMin()
	if diff != 1 {
		t.Fatalf("difficulty expected 1, got %d", diff)
	}
}

func newTestTransaction(id ids.ID, difficulty uint64, blockID ids.ID, sig []byte) *Transaction {
	return &Transaction{
		Unsigned:   newTestUnsigned(id, difficulty, blockID),
		Signature:  sig,
		difficulty: difficulty,
	}
}

func newTestUnsigned(id ids.ID, difficulty uint64, blockID ids.ID) Unsigned {
	return &testUnsigned{
		id:         id,
		difficulty: difficulty,
		blockID:    blockID,
	}
}

type testUnsigned struct {
	id         ids.ID
	difficulty uint64
	blockID    ids.ID
}

func (utx *testUnsigned) ID() ids.ID                          { return utx.id }
func (utx *testUnsigned) Difficulty() uint64                  { return utx.difficulty }
func (utx *testUnsigned) Bytes() [32]byte                     { return ids.Empty }
func (utx *testUnsigned) GetBlockID() ids.ID                  { return utx.blockID }
func (utx *testUnsigned) SetBlockID(block ids.ID)             { utx.blockID = block }
func (utx *testUnsigned) SetGraffiti(graffiti []byte)         {}
func (utx *testUnsigned) GetSender() ed25519.PublicKey        { return nil }
func (utx *testUnsigned) GetPrefix() []byte                   { return nil }
func (utx *testUnsigned) VerifyBase() error                   { return nil }
func (utx *testUnsigned) Verify(storage.Storage, int64) error { return nil }
func (utx *testUnsigned) Accept(storage.Storage, int64) error { return nil }
