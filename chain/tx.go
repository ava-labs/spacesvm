// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/spacesvm/tdata"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Transaction struct {
	UnsignedTransaction `serialize:"true" json:"unsignedTransaction"`
	Signature           []byte `serialize:"true" json:"signature"`

	digestHash []byte
	bytes      []byte
	id         ids.ID
	size       uint64
	sender     common.Address
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

func DigestHash(utx UnsignedTransaction) ([]byte, error) {
	return tdata.DigestHash(utx.TypedData())
}

func (t *Transaction) Init(g *Genesis) error {
	stx, err := Marshal(t)
	if err != nil {
		return err
	}
	t.bytes = stx

	id, err := ids.ToID(crypto.Keccak256(t.bytes))
	if err != nil {
		return err
	}
	t.id = id

	// Compute digest hash
	dh, err := DigestHash(t.UnsignedTransaction)
	if err != nil {
		return err
	}
	t.digestHash = dh

	// Derive sender
	pk, err := DeriveSender(t.digestHash, t.Signature)
	if err != nil {
		return err
	}
	t.sender = crypto.PubkeyToAddress(*pk)

	t.size = uint64(len(t.Bytes()))
	return nil
}

func (t *Transaction) Bytes() []byte { return t.bytes }

func (t *Transaction) Size() uint64 { return t.size }

func (t *Transaction) ID() ids.ID { return t.id }

func (t *Transaction) DigestHash() []byte { return t.digestHash }

func (t *Transaction) Sender() common.Address { return t.sender }

func (t *Transaction) Execute(g *Genesis, db database.Database, blk *StatelessBlock, context *Context) error {
	if err := t.UnsignedTransaction.ExecuteBase(g); err != nil {
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

	// Ensure sender has balance
	if _, err := ModifyBalance(db, t.sender, false, t.FeeUnits(g)*t.GetPrice()); err != nil {
		return err
	}
	if t.GetPrice() < context.NextPrice {
		return ErrInsufficientPrice
	}
	if err := t.UnsignedTransaction.Execute(&TransactionContext{
		Genesis:   g,
		Database:  db,
		BlockTime: uint64(blk.Tmstmp),
		TxID:      t.id,
		Sender:    t.sender,
	}); err != nil {
		return err
	}
	if err := SetTransaction(db, t); err != nil {
		return err
	}

	// Process lottery reward
	//
	// If there is no space after the selected iterator, no reward will be
	// distributed.
	if blk.Dummy() {
		// Do not process any rewards if it is just a dummy block
		return nil
	}
	rewardAmount := t.FeeUnits(g) * blk.Price * g.LotteryRewardMultipler / LotteryRewardDivisor
	if rewardAmount == 0 {
		// For transactions (like transfers) where the [FeeUnits] are equal to the [BaseTxFee], it
		// is possible that the reward could be 0.
		return nil
	}
	recipient, distributed, err := ApplyReward(db, blk.ID(), t.ID(), t.sender, rewardAmount)
	if err != nil {
		return err
	}
	if distributed {
		blk.Winners[t.ID()] = &Activity{
			Tmstmp: blk.Tmstmp,
			Typ:    Reward,
			TxID:   t.ID(),
			To:     recipient.Hex(),
			Units:  rewardAmount,
		}
	}
	return nil
}

func (t *Transaction) Activity() *Activity {
	activity := t.UnsignedTransaction.Activity()
	activity.Sender = t.sender.Hex()
	activity.TxID = t.id
	return activity
}
