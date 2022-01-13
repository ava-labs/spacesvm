// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

type Transaction struct {
	UnsignedTransaction `serialize:"true" json:"unsignedTransaction"`
	Signature           []byte `serialize:"true" json:"signature"`

	unsignedBytes []byte
	bytes         []byte
	id            ids.ID
	size          uint64
	sender        common.Address
}

func NewTx(utx UnsignedTransaction, sig []byte) *Transaction {
	return &Transaction{
		UnsignedTransaction: utx,
		Signature:           sig,
	}
}

func (t *Transaction) Copy() *Transaction {
	sig := make([]byte, len(t.Signature))
	copy(sig, t.Signature)
	return &Transaction{
		UnsignedTransaction: t.UnsignedTransaction.Copy(),
		Signature:           sig,
	}
}

func UnsignedBytes(utx UnsignedTransaction) ([]byte, error) {
	// TODO: change to human readable
	b, err := Marshal(utx)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (t *Transaction) Init(g *Genesis) error {
	utx, err := UnsignedBytes(t.UnsignedTransaction)
	if err != nil {
		return err
	}
	t.unsignedBytes = utx

	stx, err := Marshal(t)
	if err != nil {
		return err
	}
	t.bytes = stx

	h := hashing.ComputeHash256(t.bytes)
	id, err := ids.ToID(h)
	if err != nil {
		return err
	}
	t.id = id

	// Extract address
	cpk, err := secp256k1.RecoverPubkey(t.UnsignedBytes(), t.Signature)
	if err != nil {
		return err
	}
	pk, err := crypto.DecompressPubkey(cpk)
	if err != nil {
		return err
	}
	t.sender = crypto.PubkeyToAddress(*pk)

	t.size = uint64(len(t.Bytes()))
	return nil
}

func (t *Transaction) Bytes() []byte { return t.bytes }

func (t *Transaction) UnsignedBytes() []byte { return t.unsignedBytes }

func (t *Transaction) Size() uint64 { return t.size }

func (t *Transaction) ID() ids.ID { return t.id }

func (t *Transaction) Execute(g *Genesis, db database.Database, blockTime int64, context *Context) error {
	if err := t.UnsignedTransaction.ExecuteBase(g); err != nil {
		return err
	}
	if !context.RecentBlockIDs.Contains(t.BlockID()) {
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

	// Modify sender has balance
	if _, err := ModifyBalance(db, t.sender, false, t.FeeUnits(g)*t.Price()); err != nil {
		return err
	}
	if t.Price() < context.NextPrice {
		return ErrInsufficientPrice
	}
	if err := t.UnsignedTransaction.Execute(g, db, uint64(blockTime), t.id); err != nil {
		return err
	}
	return SetTransaction(db, t)
}
