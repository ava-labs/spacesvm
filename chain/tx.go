package chain

import (
	"errors"

	"ekyu.moe/cryptonight"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"

	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/crypto"
)

func init() {
	codec.RegisterType(&Transaction{})
}

type UnsignedTransaction interface {
	SetBlockID(block ids.ID)
	SetGraffiti(graffiti []byte)
	GetSender() *crypto.PublicKey
	GetBlockID() ids.ID
	VerifyBase() error
	Verify(DB, int64) error
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
		h := cryptonight.Sum(UnsignedBytes(t.UnsignedTransaction), 2)
		t.difficulty = cryptonight.Difficulty(h)
	}
	return t.difficulty
}

func (t *Transaction) Verify(db DB, blockTime int64, recentBlockIDs ids.Set, recentTxIDs ids.Set, minDifficulty uint64) error {
	if err := t.UnsignedTransaction.VerifyBase(); err != nil {
		return err
	}
	if !recentBlockIDs.Contains(t.GetBlockID()) {
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
	if !t.GetSender().Verify(UnsignedBytes(t.UnsignedTransaction), t.Signature) {
		return errors.New("invalid signature")
	}
	if err := t.UnsignedTransaction.Verify(db, blockTime); err != nil {
		return err
	}
	return db.StoreTransaction(t)
}