// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"errors"
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestTransferTx(t *testing.T) {
	t.Parallel()

	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	sender := crypto.PubkeyToAddress(priv.PublicKey)

	priv2, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	sender2 := crypto.PubkeyToAddress(priv2.PublicKey)

	priv3, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	sender3 := crypto.PubkeyToAddress(priv3.PublicKey)

	db := memdb.New()
	defer db.Close()

	g := DefaultGenesis()
	g.CustomAllocation = []*CustomAllocation{
		{
			Address: sender,
			Balance: 10000000,
		},
		{
			Address: sender2,
			Balance: 1,
		},
		// sender3 is not given any balance
	}
	if err := g.Load(db, nil); err != nil {
		t.Fatal(err)
	}

	// Items: transfer without balance, transfer with small balance, transfer some balance, transfer from
	// account that now has balance, transfer space, transfer to self
	tt := []struct {
		utx       UnsignedTransaction
		blockTime uint64
		sender    common.Address
		err       error
	}{
		{ // invalid when no amount is given
			utx:       &TransferTx{BaseTx: &BaseTx{}},
			blockTime: 1,
			sender:    sender,
			err:       ErrNonActionable,
		},
		{ // invalid when no funds
			utx:       &TransferTx{BaseTx: &BaseTx{}, To: sender, Units: 10},
			blockTime: 1,
			sender:    sender3,
			err:       ErrInvalidBalance,
		},
		{ // invalid when little funds
			utx:       &TransferTx{BaseTx: &BaseTx{}, To: sender, Units: 10},
			blockTime: 1,
			sender:    sender2,
			err:       ErrInvalidBalance,
		},
		{ // valid send to new account
			utx:       &TransferTx{BaseTx: &BaseTx{}, To: sender3, Units: 10},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // invalid send to no one
			utx:       &TransferTx{BaseTx: &BaseTx{}, Units: 10},
			blockTime: 1,
			sender:    sender,
			err:       ErrNonActionable,
		},
		{ // valid send to existing account
			utx:       &TransferTx{BaseTx: &BaseTx{}, To: sender2, Units: 10},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // now valid
			utx:       &TransferTx{BaseTx: &BaseTx{}, To: sender, Units: 10},
			blockTime: 1,
			sender:    sender3,
			err:       nil,
		},
		{ // now valid
			utx:       &TransferTx{BaseTx: &BaseTx{}, To: sender, Units: 10},
			blockTime: 1,
			sender:    sender2,
			err:       nil,
		},
	}
	for i, tv := range tt {
		tc := &TransactionContext{
			Genesis:   g,
			Database:  db,
			BlockTime: tv.blockTime,
			TxID:      ids.Empty,
			Sender:    tv.sender,
		}
		err := tv.utx.Execute(tc)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.err, err)
		}
	}
}
