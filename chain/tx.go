package chain

import (
	"ekyu.moe/cryptonight"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"

	"github.com/ava-labs/quarkvm/crypto"
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

	difficulty uint64 // populate in mempool
}

func NewTx(utx UnsignedTransaction, sig []byte) *Transaction {
	return &Transaction{
		UnsignedTransaction: utx,
		Signature:           sig,
	}
}

func UnsignedBytes(utx UnsignedTransaction) []byte {
	v, err := Marshal(utx)
	if err != nil {
		panic(err)
	}
	return v
}

func (t *Transaction) Bytes() []byte {
	v, err := Marshal(t)
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
		h := cryptonight.Sum(UnsignedBytes(t.UnsignedTransaction), 2)
		t.difficulty = cryptonight.Difficulty(h)
	}
	return t.difficulty
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
	if t.Difficulty() < context.NextDifficulty {
		return ErrInvalidDifficulty
	}
	if !crypto.Verify(t.GetSender(), UnsignedBytes(t.UnsignedTransaction), t.Signature) {
		return ErrInvalidSignature
	}
	if err := t.UnsignedTransaction.Execute(db, blockTime); err != nil {
		return err
	}
	return SetTransaction(db, t)
}
