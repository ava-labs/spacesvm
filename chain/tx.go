// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"

	"github.com/ava-labs/quarkvm/crypto"
	"github.com/ava-labs/quarkvm/pow"
)

type UnsignedTransaction interface {
	SetBlockID(block ids.ID)
	SetGraffiti(graffiti uint64)
	GetSender() [crypto.PublicKeySize]byte
	GetBlockID() ids.ID

	ExecuteBase() error
	Execute(database.Database, int64) error
}

type Transaction struct {
	UnsignedTransaction `serialize:"true"`
	Signature           []byte `serialize:"true"`

	unsignedBytes []byte
	bytes         []byte
	id            ids.ID
	size          uint64
	difficulty    uint64
}

func NewTx(utx UnsignedTransaction, sig []byte) *Transaction {
	return &Transaction{
		UnsignedTransaction: utx,
		Signature:           sig,
	}
}

func UnsignedBytes(utx UnsignedTransaction) ([]byte, error) {
	b, err := Marshal(utx)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (t *Transaction) Init() error {
	utx, err := UnsignedBytes(t.UnsignedTransaction)
	if err != nil {
		return err
	}
	t.unsignedBytes = utx
	t.difficulty = pow.Difficulty(utx)

	stx, err := Marshal(t)
	if err != nil {
		return err
	}
	t.bytes = stx

	id, err := ids.ToID(hashing.ComputeHash256(t.bytes))
	if err != nil {
		return err
	}
	t.id = id

	t.size = uint64(len(t.Bytes()))
	return nil
}

func (t *Transaction) Bytes() []byte { return t.bytes }

func (t *Transaction) UnsignedBytes() []byte { return t.unsignedBytes }

func (t *Transaction) Size() uint64 { return t.size }

func (t *Transaction) ID() ids.ID { return t.id }

func (t *Transaction) Difficulty() uint64 { return t.difficulty }

func (t *Transaction) Execute(db database.Database, blockTime int64, context *Context) error {
	if err := t.UnsignedTransaction.ExecuteBase(); err != nil {
		return err
	}
	if !context.RecentBlockIDs.Contains(t.GetBlockID()) {
		// Hash must be recent to be any good
		// Should not happen beause of mempool cleanup
		return ErrInvalidBlockID
	}
	if context.RecentTxIDs.Contains(t.ID()) {
		// Tx hash must not be recently executed (otherwise could be replayed)
		//
		// NOTE: We only need to keep cached tx hashes around as long as the
		// block hash referenced in the tx is valid
		return ErrDuplicateTx
	}
	if t.Difficulty() < context.NextDifficulty {
		return ErrInvalidDifficulty
	}
	if !crypto.Verify(t.GetSender(), t.unsignedBytes, t.Signature) {
		return ErrInvalidSignature
	}
	if err := t.UnsignedTransaction.Execute(db, blockTime); err != nil {
		return err
	}
	return SetTransaction(db, t)
}
