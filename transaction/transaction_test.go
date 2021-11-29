// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package transaction

import (
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/quarkvm/crypto/ed25519"
	"github.com/ava-labs/quarkvm/storage"
)

func TestTransactionCodec(t *testing.T) {
	tx := &Transaction{
		Unsigned: Unsigned{
			PublicKey: []byte("123"),
		},
		Signature: []byte("1"),
	}

	txID := tx.ID()

	// difficulty field should be ignored for tx ID computation
	tx.difficulty = 1
	if txID != tx.ID() {
		t.Fatalf("txID expected %v, got %v", txID, tx.ID())
	}

	// should change the ID
	tx.Unsigned.Key = "foo"
	txID2 := tx.ID()

	if txID != txID2 {
		t.Fatalf("txID expected %v, got %v", txID, txID2)
	}
}

func TestTransactionErrInvalidSig(t *testing.T) {
	priv, err := ed25519.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.PublicKey()

	tx := &Transaction{
		Unsigned: Unsigned{
			PublicKey: pub.Bytes(),
			Op:        "Put",
			Key:       "foo",
			Value:     "bar",
		},
	}
	tx.Signature, err = priv.Sign(tx.Unsigned.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	if err := tx.Verify(); err != nil {
		t.Fatal(err)
	}
	tx.Signature = []byte("invalid")
	if err := tx.Verify(); err != ErrInvalidSig {
		t.Fatalf("expected %v, got %v", ErrInvalidSig, err)
	}
}

func TestTransactionAccept(t *testing.T) {
	s := storage.New(snow.DefaultContextTest(), memdb.New())
	defer s.Close()

	tx := &Transaction{
		Unsigned: Unsigned{
			PublicKey: []byte("123"),
			Op:        "Put",
			Key:       "foo",
			Value:     "bar",
		},
		Signature: []byte("1"),
	}

	txID := tx.ID()

	if err := tx.Accept(s, 1); err != nil {
		t.Fatal(err)
	}
	if has, err := s.Tx().Has(txID[:]); !has || err != nil {
		t.Fatalf("s.Tx().Has unexpected has %v or err %v", has, err)
	}
	if has, err := s.Key().Has([]byte(tx.Unsigned.Key)); !has || err != nil {
		t.Fatalf("s.Key().Has unexpected has %v or err %v", has, err)
	}
}
