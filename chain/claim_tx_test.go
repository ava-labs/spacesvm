// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestClaimTx(t *testing.T) {
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

	db := memdb.New()
	defer db.Close()

	g := DefaultGenesis()
	ClaimReward := int64(g.ClaimReward)
	tt := []struct {
		tx        *ClaimTx
		blockTime int64
		sender    common.Address
		err       error
	}{
		{ // invalid claim, [42]byte space is reserved for pubkey
			tx:        &ClaimTx{BaseTx: &BaseTx{}, Space: strings.Repeat("a", hexAddressLen)},
			blockTime: 1,
			sender:    sender,
			err:       ErrAddressMismatch,
		},
		{ // valid claim, [42]byte space is reserved for pubkey
			tx:        &ClaimTx{BaseTx: &BaseTx{}, Space: strings.ToLower(sender.Hex())},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // successful claim with expiry time "blockTime" + "expiryTime"
			tx:        &ClaimTx{BaseTx: &BaseTx{}, Space: "foo"},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // invalid claim due to expiration
			tx:        &ClaimTx{BaseTx: &BaseTx{}, Space: "foo"},
			blockTime: 100,
			sender:    sender,
			err:       ErrSpaceNotExpired,
		},
		{ // successful new claim
			tx:        &ClaimTx{BaseTx: &BaseTx{}, Space: "foo"},
			blockTime: ClaimReward * 2,
			sender:    sender,
			err:       nil,
		},
		{ // successful new claim by different owner
			tx:        &ClaimTx{BaseTx: &BaseTx{}, Space: "foo"},
			blockTime: ClaimReward * 4,
			sender:    sender2,
			err:       nil,
		},
		{ // invalid claim due to expiration by different owner
			tx:        &ClaimTx{BaseTx: &BaseTx{}, Space: "foo"},
			blockTime: ClaimReward*4 + 3,
			sender:    sender2,
			err:       ErrSpaceNotExpired,
		},
	}
	for i, tv := range tt {
		if i > 0 {
			// Expire old spaces between txs
			if err := ExpireNext(db, tt[i-1].blockTime, tv.blockTime, true); err != nil {
				t.Fatalf("#%d: ExpireNext errored %v", i, err)
			}
		}
		tc := &TransactionContext{
			Genesis:   g,
			Database:  db,
			BlockTime: uint64(tv.blockTime),
			TxID:      ids.Empty,
			Sender:    tv.sender,
		}
		err := tv.tx.Execute(tc)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.err, err)
		}
		if tv.err != nil {
			continue
		}
		info, exists, err := GetSpaceInfo(db, []byte(tv.tx.Space))
		if err != nil {
			t.Fatalf("#%d: failed to get space info %v", i, err)
		}
		if !exists {
			t.Fatalf("#%d: failed to find space info", i)
		}
		if !bytes.Equal(info.Owner[:], tv.sender[:]) {
			t.Fatalf("#%d: unexpected owner found (expected pub key %q)", i, string(sender[:]))
		}
	}

	// Cleanup DB after all txs submitted
	senderSpaces, err := GetAllOwned(db, sender)
	if err != nil {
		t.Fatal(err)
	}
	if len(senderSpaces) != 0 {
		t.Fatalf("sender owned spaces should = 0, found %d", len(senderSpaces))
	}
	sender2Spaces, err := GetAllOwned(db, sender2)
	if err != nil {
		t.Fatal(err)
	}
	if len(sender2Spaces) != 1 {
		t.Fatalf("sender2 owned spaces should = 1, found %d", len(sender2Spaces))
	}
	if err := ExpireNext(db, 0, ClaimReward*10, true); err != nil {
		t.Fatal(err)
	}
	pruned, err := PruneNext(db, 100)
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 4 {
		t.Fatalf("expected to prune 4 but got %d", pruned)
	}
	_, exists, err := GetSpaceInfo(db, []byte("foo"))
	if err != nil {
		t.Fatalf("failed to get space info %v", err)
	}
	if exists {
		t.Fatal("space should not exist")
	}
	senderSpaces, err = GetAllOwned(db, sender)
	if err != nil {
		t.Fatal(err)
	}
	if len(senderSpaces) != 0 {
		t.Fatal("owned spaces should be empty")
	}
	sender2Spaces, err = GetAllOwned(db, sender2)
	if err != nil {
		t.Fatal(err)
	}
	if len(sender2Spaces) != 0 {
		t.Fatal("owned spaces should be empty")
	}
}
