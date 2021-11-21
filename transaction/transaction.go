// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package transaction defines the transaction interface and objects.
package transaction

import (
	"errors"

	"ekyu.moe/cryptonight"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/storage"
)

func init() {
	codec.RegisterType(&Transaction{})
}

type Transaction struct {
	Unsigned  Unsigned `serialize:"true" json:"unsigned"`
	Signature []byte   `serialize:"true" json:"signature"`

	difficulty uint64 // populate in mempool
}

func New(utx Unsigned, sig []byte) *Transaction {
	return &Transaction{
		Unsigned:  utx,
		Signature: sig,
	}
}

func UnsignedBytes(utx Unsigned) []byte {
	v, err := codec.Marshal(utx)
	if err != nil {
		panic(err)
	}
	return v
}

func (t *Transaction) Bytes() []byte {
	v, err := codec.Marshal(t)
	if err != nil {
		panic(err)
	}
	return v
}

func (t *Transaction) Size() uint64 {
	return uint64(len(t.Bytes()))
}

func (t *Transaction) ID() ids.ID {
	h, err := ids.ToID(hashing.ComputeHash256(t.Bytes()))
	if err != nil {
		panic(err)
	}
	return h
}

func (t *Transaction) Difficulty() uint64 {
	if t.difficulty == 0 {
		h := cryptonight.Sum(UnsignedBytes(t.Unsigned), 2)
		t.difficulty = cryptonight.Difficulty(h)
	}
	return t.difficulty
}

func (t *Transaction) Verify(s storage.Storage, blockTime int64, recentBlockIDs ids.Set, recentTxIDs ids.Set, minDifficulty uint64) error {
	if err := t.Unsigned.VerifyBase(); err != nil {
		return err
	}
	if !recentBlockIDs.Contains(t.Unsigned.GetBlockID()) {
		// Hash must be recent to be any good
		// Should not happen beause of mempool cleanup
		return errors.New("invalid block id")
	}
	if recentTxIDs.Contains(t.ID()) {
		// Tx hash must not be recently executed (otherwise could be replayed)
		//
		// NOTE: We only need to keep cached tx hashes around as long as the
		// block hash referenced in the tx is valid
		return errors.New("duplicate tx")
	}
	if t.Difficulty() < minDifficulty {
		return errors.New("invalid difficulty")
	}
	if !t.Unsigned.GetSender().Verify(UnsignedBytes(t.Unsigned), t.Signature) {
		return errors.New("invalid signature")
	}
	return t.Unsigned.Verify(s, blockTime)
}

func (t *Transaction) Accept(s storage.Storage, blockTime int64) error {
	if err := t.Unsigned.Accept(s, blockTime); err != nil {
		return err
	}

	// persists in prefixed db
	id := t.ID()
	return s.Tx().Put(append([]byte{}, id[:]...), nil)
}

func (t *Transaction) PrefixID() ids.ID {
	h, err := ids.ToID(hashing.ComputeHash256(t.Unsigned.GetPrefix()))
	if err != nil {
		panic(err)
	}
	return h
}
