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

func TestLifelineTx(t *testing.T) {
	t.Parallel()

	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	sender := crypto.PubkeyToAddress(priv.PublicKey)

	db := memdb.New()
	defer db.Close()

	g := DefaultGenesis()
	tt := []struct {
		utx       UnsignedTransaction
		blockTime uint64
		sender    common.Address
		err       error
	}{
		{ // invalid when prefix info is missing
			utx:       &LifelineTx{BaseTx: &BaseTx{Pfx: []byte("foo")}},
			blockTime: 1,
			sender:    sender,
			err:       ErrPrefixMissing,
		},
		{ // successful claim with expiry time "blockTime" + "expiryTime"
			utx:       &ClaimTx{BaseTx: &BaseTx{Pfx: []byte("foo")}},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // successful lifeline when prefix info is not missing
			utx:       &LifelineTx{BaseTx: &BaseTx{Pfx: []byte("foo")}},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
	}
	for i, tv := range tt {
		tc := &TransactionContext{
			Genesis:   g,
			Database:  db,
			BlockTime: uint64(tv.blockTime),
			TxID:      ids.Empty,
			Sender:    tv.sender,
		}
		err := tv.utx.Execute(tc)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.err, err)
		}
	}
}
