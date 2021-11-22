// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package transaction defines the transaction interface and objects.
package transaction

import (
	"errors"
	"fmt"

	"ekyu.moe/cryptonight"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/crypto/ed25519"
	"github.com/ava-labs/quarkvm/storage"
)

var ErrInvalidSig = errors.New("invalid signature")

func init() {
	codec.RegisterType(&Transaction{})
}

func New(utx Unsigned, sig []byte) *Transaction {
	return &Transaction{
		Unsigned:  utx,
		Signature: sig,
	}
}

type Transaction struct {
	Unsigned  Unsigned `serialize:"true" json:"unsigned"`
	Signature []byte   `serialize:"true" json:"signature"`

	// TODO: using this?
	Encoding formatting.Encoding `serialize:"true" json:"encoding"`

	difficulty uint64 `serialize:"false" json:"-"`
	txID       ids.ID `serialize:"false" json:"-"`
}

func (t *Transaction) Bytes() []byte {
	v, err := codec.Marshal(t)
	if err != nil {
		panic(err)
	}
	return v
}

func (t *Transaction) ID() ids.ID {
	if t.txID == ids.Empty {
		h, err := ids.ToID(hashing.ComputeHash256(t.Bytes()))
		if err != nil {
			panic(err)
		}
		t.txID = h
	}
	return t.txID
}

func (t *Transaction) Difficulty() uint64 {
	if t.difficulty == 0 {
		h := cryptonight.Sum(t.Unsigned.Bytes(), 2)
		t.difficulty = cryptonight.Difficulty(h)
	}
	return t.difficulty
}

func (t *Transaction) PrefixID() ids.ID {
	pfx, err := t.Unsigned.GetPrefix()
	if err != nil {
		panic(err)
	}
	h, err := ids.ToID(hashing.ComputeHash256(pfx))
	if err != nil {
		panic(err)
	}
	return h
}

func (t *Transaction) Verify() error {
	switch t.Unsigned.Op {
	case "Put":
	case "Range":
	default:
		return fmt.Errorf("unknown op %q", t.Unsigned.Op)
	}
	if !ed25519.Verify(t.Unsigned.PublicKey, t.Unsigned.Bytes(), t.Signature) {
		return ErrInvalidSig
	}
	return nil
}

func (t *Transaction) Accept(s storage.Storage, blockTime int64) error {
	// persist to database once PoW/agreed by consensus
	switch t.Unsigned.Op {
	case "Put":
		// persist key-value pair
		if err := s.Put(
			[]byte(t.Unsigned.Key),
			[]byte(t.Unsigned.Value),
			storage.WithOverwrite(true),
			storage.WithBlockTime(blockTime),
			storage.WithPublicKey(&ed25519.PublicKey{PublicKey: []byte(t.Unsigned.PublicKey)}),
		); err != nil {
			return err
		}
		// if key-value write succeeds,
		// persists the transaction ID
		txID := t.ID()
		if err := s.Tx().Put(txID[:], nil); err != nil {
			return err
		}

	case "Range":
		// no-op for accept
	}
	return nil
}
