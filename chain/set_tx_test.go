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

	"github.com/ava-labs/spacesvm/parser"
)

func TestSetTx(t *testing.T) {
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
	tt := []struct {
		utx       UnsignedTransaction
		blockTime int64
		sender    common.Address
		err       error
	}{
		{ // write with no previous claim should fail
			utx: &SetTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   "bar",
				Value: []byte("value"),
			},
			blockTime: 1,
			sender:    sender,
			err:       ErrSpaceMissing,
		},
		{ // successful claim
			utx: &ClaimTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
			},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // write
			utx: &SetTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   "bar",
				Value: []byte("value"),
			},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // write empty
			utx: &SetTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   "bar",
			},
			blockTime: 1,
			sender:    sender,
			err:       ErrValueEmpty,
		},
		{ // delete
			utx: &DeleteTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   "bar",
			},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // write hashed value
			utx: &SetTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   valueHash([]byte("value")),
				Value: []byte("value"),
			},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // write incorrect hashed value
			utx: &SetTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   valueHash([]byte("not value")),
				Value: []byte("value"),
			},
			blockTime: 1,
			sender:    sender,
			err:       ErrInvalidKey,
		},
		{ // delete hashed value
			utx: &DeleteTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   valueHash([]byte("value")),
			},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // delete incorrect hashed value
			utx: &DeleteTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   valueHash([]byte("not value")),
			},
			blockTime: 1,
			sender:    sender,
			err:       ErrKeyMissing,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   "bar",
				Value: []byte("value"),
			},
			blockTime: 1,
			sender:    sender2,
			err:       ErrUnauthorized,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
			},
			blockTime: 1,
			err:       parser.ErrInvalidContents,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   strings.Repeat("a", parser.MaxIdentifierSize+1),
			},
			blockTime: 1,
			err:       parser.ErrInvalidContents,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   "bar",
				Value: bytes.Repeat([]byte{'b'}, int(g.MaxValueSize)+1),
			},
			blockTime: 1,
			err:       ErrValueTooBig,
		},
		{
			utx: &DeleteTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   "bar",
			},
			blockTime: 100,
			sender:    sender,
			err:       ErrKeyMissing,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   "bar",
				Value: []byte("value"),
			},
			blockTime: int64(g.ClaimReward) * 2,
			sender:    sender,
			err:       ErrSpaceMissing,
		},
	}
	for i, tv := range tt {
		if i > 0 {
			// Expire old spaces between txs
			if err := ExpireNext(db, tt[i-1].blockTime, tv.blockTime, true); err != nil {
				t.Fatalf("#%d: ExpireNext errored %v", i, err)
			}
		}
		// Set linked value (normally done in block processing)
		id := ids.GenerateTestID()
		if tp, ok := tv.utx.(*SetTx); ok {
			if len(tp.Value) > 0 {
				if err := db.Put(PrefixTxValueKey(id), tp.Value); err != nil {
					t.Fatal(err)
				}
			}
		}
		tc := &TransactionContext{
			Genesis:   g,
			Database:  db,
			BlockTime: uint64(tv.blockTime),
			TxID:      id,
			Sender:    tv.sender,
		}
		err := tv.utx.Execute(tc)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.err, err)
		}
		if tv.err != nil {
			continue
		}

		// check committed states from db
		switch tp := tv.utx.(type) {
		case *ClaimTx: // "ClaimTx.Execute" must persist "SpaceInfo"
			info, exists, err := GetSpaceInfo(db, []byte(tp.Space))
			if err != nil {
				t.Fatalf("#%d: failed to get space info %v", i, err)
			}
			if !exists {
				t.Fatalf("#%d: failed to find space info", i)
			}
			if !bytes.Equal(info.Owner[:], tv.sender[:]) {
				t.Fatalf("#%d: unexpected owner found (expected pub key %q)", i, string(sender[:]))
			}
		case *SetTx:
			vmeta, exists, err := GetValueMeta(db, []byte(tp.Space), []byte(tp.Key))
			if err != nil {
				t.Fatalf("#%d: failed to get meta info %v", i, err)
			}
			switch {
			case !exists:
				t.Fatalf("#%d: non-empty value should have been persisted but not found", i)
			case exists:
				if vmeta.TxID != id {
					t.Fatalf("#%d: unexpected txID %q, expected %q", i, vmeta.TxID, id)
				}
			}

			val, exists, err := GetValue(db, []byte(tp.Space), []byte(tp.Key))
			if err != nil {
				t.Fatalf("#%d: failed to get key info %v", i, err)
			}
			switch {
			case !exists:
				t.Fatalf("#%d: non-empty value should have been persisted but not found", i)
			case exists:
				if !bytes.Equal(tp.Value, val) {
					t.Fatalf("#%d: unexpected value %q, expected %q", i, val, tp.Value)
				}
			}
		case *DeleteTx:
			_, exists, err := GetValue(db, []byte(tp.Space), []byte(tp.Key))
			if err != nil {
				t.Fatalf("#%d: failed to get key info %v", i, err)
			}
			if exists {
				t.Fatalf("#%d: empty value should have deleted keys", i)
			}
		}
	}
}
