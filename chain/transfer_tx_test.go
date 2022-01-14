// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"errors"
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/spacesvm/parser"
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
	g.Allocations = []*Allocation{
		{
			Address: sender.Hex(),
			Balance: 10000000,
		},
		{
			Address: sender2.Hex(),
			Balance: 1,
		},
		// sender3 is not given any balance
	}
	if err := g.Load(db); err != nil {
		t.Fatal(err)
	}

	// Items: transfer without balance, transfer with small balance, transfer some balance, transfer from
	// account that now has balance, transfer prefix, transfer to self
	tt := []struct {
		utx       UnsignedTransaction
		blockTime uint64
		sender    common.Address
		err       error
	}{
		{ // invalid when prefix info is missing
			utx:       &TransferTx{BaseTx: &BaseTx{Pfx: []byte("foo")}},
			blockTime: 1,
			sender:    sender,
			err:       ErrPrefixMissing,
		},
		{ // invalid when no prefix or amount is given
			utx:       &TransferTx{BaseTx: &BaseTx{}},
			blockTime: 1,
			sender:    sender,
			err:       ErrNonActionable,
		},
		{ // invalid when no funds
			utx:       &TransferTx{BaseTx: &BaseTx{}, To: sender, Value: 10},
			blockTime: 1,
			sender:    sender3,
			err:       ErrInvalidBalance,
		},
		{ // invalid when little funds
			utx:       &TransferTx{BaseTx: &BaseTx{}, To: sender, Value: 10},
			blockTime: 1,
			sender:    sender2,
			err:       ErrInvalidBalance,
		},
		{ // valid send to new account
			utx:       &TransferTx{BaseTx: &BaseTx{}, To: sender3, Value: 10},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // valid send to existing account
			utx:       &TransferTx{BaseTx: &BaseTx{}, To: sender2, Value: 10},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // now valid
			utx:       &TransferTx{BaseTx: &BaseTx{}, To: sender, Value: 10},
			blockTime: 1,
			sender:    sender3,
			err:       nil,
		},
		{ // now valid
			utx:       &TransferTx{BaseTx: &BaseTx{}, To: sender, Value: 10},
			blockTime: 1,
			sender:    sender2,
			err:       nil,
		},
		{ // successful claim with expiry time "blockTime" + "expiryTime"
			utx:       &ClaimTx{BaseTx: &BaseTx{Pfx: []byte("foo")}},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // successful prefix transfer
			utx:       &TransferTx{BaseTx: &BaseTx{Pfx: []byte("foo")}, To: sender3},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // prefix looking bad
			utx:       &TransferTx{BaseTx: &BaseTx{Pfx: []byte("foo/")}, To: sender3},
			blockTime: 1,
			sender:    sender,
			err:       parser.ErrInvalidDelimiter,
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
