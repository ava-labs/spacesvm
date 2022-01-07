// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"encoding/binary"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"golang.org/x/crypto/sha3"

	"github.com/ava-labs/quarkvm/pow"
)

type MinedTransaction struct {
	UnsignedTransaction `serialize:"true" json:"unsignedTransaction"`
	Graffiti            []uint64 `serialize:"true" json:"graffiti"`
}

type Transaction struct {
	*MinedTransaction `serialize:"true" json:"minedTransaction"`
	Signature         []byte `serialize:"true" json:"signature"`

	unsignedBytes []byte
	minedBytes    []byte
	bytes         []byte
	id            ids.ID
	size          uint64
	difficulty    []uint64
}

func NewMinedTx(utx UnsignedTransaction, grf []uint64) *MinedTransaction {
	return &MinedTransaction{
		UnsignedTransaction: utx,
		Graffiti:            grf,
	}
}

func NewTx(mtx *MinedTransaction, sig []byte) *Transaction {
	return &Transaction{
		MinedTransaction: mtx,
		Signature:        sig,
	}
}

func UnsignedBytes(utx UnsignedTransaction) ([]byte, error) {
	b, err := Marshal(utx)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func MinedBytes(mtx *MinedTransaction) ([]byte, error) {
	b, err := Marshal(mtx)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func HashableBytes(utx []byte, g uint64) []byte {
	unsignedLen := len(utx)
	b := make([]byte, unsignedLen+8)
	copy(b, utx)
	binary.LittleEndian.PutUint64(b[unsignedLen:], g)
	return b
}

func (t *Transaction) Init() error {
	utx, err := UnsignedBytes(t.UnsignedTransaction)
	if err != nil {
		return err
	}
	t.unsignedBytes = utx

	mtx, err := MinedBytes(t.MinedTransaction)
	if err != nil {
		return err
	}
	t.minedBytes = mtx

	t.difficulty = make([]uint64, len(t.Graffiti))
	for i, g := range t.Graffiti {
		b := HashableBytes(t.unsignedBytes, g)
		t.difficulty[i] = pow.Difficulty(b)
	}

	stx, err := Marshal(t)
	if err != nil {
		return err
	}
	t.bytes = stx

	h := sha3.Sum256(t.bytes)
	id, err := ids.ToID(h[:])
	if err != nil {
		return err
	}
	t.id = id

	t.size = uint64(len(t.Bytes()))
	return nil
}

func (t *Transaction) Bytes() []byte { return t.bytes }

func (t *Transaction) UnsignedBytes() []byte { return t.unsignedBytes }

func (t *Transaction) MinedBytes() []byte { return t.minedBytes }

func (t *Transaction) Size() uint64 { return t.size }

func (t *Transaction) ID() ids.ID { return t.id }

func (t *Transaction) MinDifficulty() uint64 {
	min := uint64(0)
	set := false
	for _, d := range t.difficulty {
		if d < min || !set {
			min = d
		}
	}
	return min
}

func (t *Transaction) Work(minDifficuty uint64) uint64 {
	w := uint64(0)
	for _, d := range t.difficulty {
		if d >= minDifficuty {
			w++
		}
	}
	return w
}

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
	// TODO: probably want to move this further up to avoid unnecessary complex checking
	if int64(len(t.Graffiti)) < t.Units() {
		return ErrInvalidDifficulty
	}
	// TODO: shouldn't need to cast
	if t.Work(context.NextDifficulty) < uint64(t.Units()) {
		return ErrInvalidDifficulty
	}
	sender := t.GetSender()
	pk, err := f.ToPublicKey(sender[:])
	if err != nil {
		return err
	}
	if !pk.Verify(t.minedBytes, t.Signature) {
		return ErrInvalidSignature
	}
	if err := t.UnsignedTransaction.Execute(db, blockTime); err != nil {
		return err
	}
	return SetTransaction(db, t)
}
