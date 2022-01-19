// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/spacesvm/parser"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestSpaceValueKey(t *testing.T) {
	t.Parallel()

	tt := []struct {
		rspc     ids.ShortID
		key      []byte
		valueKey []byte
	}{
		{
			rspc:     ids.ShortID{0x1},
			key:      []byte("hello"),
			valueKey: append([]byte{keyPrefix}, []byte("/\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00/hello")...), //nolint:lll
		},
	}
	for i, tv := range tt {
		vv := SpaceValueKey(tv.rspc, tv.key)
		if !bytes.Equal(tv.valueKey, vv) {
			t.Fatalf("#%d: value expected %q, got %q", i, tv.valueKey, vv)
		}
	}
}

func TestSpaceInfoKey(t *testing.T) {
	t.Parallel()

	tt := []struct {
		spc     []byte
		infoKey []byte
	}{
		{
			spc:     []byte("foo"),
			infoKey: append([]byte{infoPrefix}, []byte("/foo")...),
		},
	}
	for i, tv := range tt {
		vv := SpaceInfoKey(tv.spc)
		if !bytes.Equal(tv.infoKey, vv) {
			t.Fatalf("#%d: value expected %q, got %q", i, tv.infoKey, vv)
		}
	}
}

func TestPrefixTxKey(t *testing.T) {
	t.Parallel()

	id := ids.GenerateTestID()
	tt := []struct {
		txID  ids.ID
		txKey []byte
	}{
		{
			txID:  id,
			txKey: append([]byte{txPrefix, parser.ByteDelimiter}, id[:]...),
		},
	}
	for i, tv := range tt {
		vv := PrefixTxKey(tv.txID)
		if !bytes.Equal(tv.txKey, vv) {
			t.Fatalf("#%d: value expected %q, got %q", i, tv.txKey, vv)
		}
	}
}

func TestPrefixBlockKey(t *testing.T) {
	t.Parallel()

	id := ids.GenerateTestID()
	tt := []struct {
		blkID    ids.ID
		blockKey []byte
	}{
		{
			blkID:    id,
			blockKey: append([]byte{blockPrefix, parser.ByteDelimiter}, id[:]...),
		},
	}
	for i, tv := range tt {
		vv := PrefixBlockKey(tv.blkID)
		if !bytes.Equal(tv.blockKey, vv) {
			t.Fatalf("#%d: value expected %q, got %q", i, tv.blockKey, vv)
		}
	}
}

func TestPutSpaceInfoAndKey(t *testing.T) {
	t.Parallel()

	db := memdb.New()
	defer db.Close()

	spc := []byte("foo")
	k, v := []byte("k"), &ValueMeta{}

	// expect failures for non-existing spaceInfo
	if ok, err := HasSpace(db, spc); ok || err != nil {
		t.Fatalf("unexpected ok %v, err %v", ok, err)
	}
	if ok, err := HasSpaceKey(db, spc, k); ok || err != nil {
		t.Fatalf("unexpected ok %v, err %v", ok, err)
	}
	if err := PutSpaceKey(db, spc, k, v); !errors.Is(err, ErrSpaceMissing) {
		t.Fatalf("unexpected error %v, expected %v", err, ErrSpaceMissing)
	}

	if err := PutSpaceInfo(
		db,
		spc,
		&SpaceInfo{
			RawSpace: ids.ShortID{0x1},
		},
		0,
	); err != nil {
		t.Fatal(err)
	}
	if err := PutSpaceKey(db, spc, k, v); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	// expect success for existing spaceInfo
	if ok, err := HasSpace(db, spc); !ok || err != nil {
		t.Fatalf("unexpected ok %v, err %v", ok, err)
	}
	if ok, err := HasSpaceKey(db, spc, k); !ok || err != nil {
		t.Fatalf("unexpected ok %v, err %v", ok, err)
	}
}

func TestSpecificTimeKey(t *testing.T) {
	rspc0 := ids.ShortID{'k'}
	k := PrefixExpiryKey(100, rspc0)
	ts, rspc, err := extractSpecificTimeKey(k)
	if err != nil {
		t.Fatal(err)
	}
	if ts != 100 {
		t.Fatalf("unexpected timestamp %d, expected 100", ts)
	}
	if rspc != rspc0 {
		t.Fatalf("unexpected rawSpace %v, expected %v", rspc, rspc0)
	}

	if _, _, err = extractSpecificTimeKey(k[:10]); !errors.Is(err, ErrInvalidKeyFormat) {
		t.Fatalf("unexpected error %v, expected %v", err, ErrInvalidKeyFormat)
	}
}

func TestGetAllValueMetas(t *testing.T) {
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
		space     string
		blockTime int64
		sender    common.Address
		expected  []*KeyValueMeta
	}{
		{ // successful claim
			utx: &ClaimTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
			},
			space:     "foo",
			blockTime: 1,
			sender:    sender,
			expected:  []*KeyValueMeta{},
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
			space:     "foo",
			blockTime: 1,
			sender:    sender,
			expected: []*KeyValueMeta{
				{
					Key: "bar",
					ValueMeta: &ValueMeta{
						Size:    5,
						Created: 1,
						Updated: 1,
					},
				},
			},
		},
		{ // successful claim
			utx: &ClaimTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo2",
			},
			space:     "foo2",
			blockTime: 1,
			sender:    sender2,
			expected:  []*KeyValueMeta{},
		},
		{ // write
			utx: &SetTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo2",
				Key:   "bar",
				Value: []byte("value2"),
			},
			space:     "foo2",
			blockTime: 1,
			sender:    sender2,
			expected: []*KeyValueMeta{
				{
					Key: "bar",
					ValueMeta: &ValueMeta{
						Size:    6,
						Created: 1,
						Updated: 1,
					},
				},
			},
		},
		{ // write again
			utx: &SetTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   "bar",
				Value: []byte("value2"),
			},
			space:     "foo",
			blockTime: 2,
			sender:    sender,
			expected: []*KeyValueMeta{
				{
					Key: "bar",
					ValueMeta: &ValueMeta{
						Size:    6,
						Created: 1,
						Updated: 2,
					},
				},
			},
		},
		{ // write new
			utx: &SetTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   "bar2",
				Value: []byte("value2"),
			},
			space:     "foo",
			blockTime: 2,
			sender:    sender,
			expected: []*KeyValueMeta{
				{
					Key: "bar",
					ValueMeta: &ValueMeta{
						Size:    6,
						Created: 1,
						Updated: 2,
					},
				},
				{
					Key: "bar2",
					ValueMeta: &ValueMeta{
						Size:    6,
						Created: 2,
						Updated: 2,
					},
				},
			},
		},
		{ // delete
			utx: &DeleteTx{
				BaseTx: &BaseTx{
					BlockID: ids.GenerateTestID(),
				},
				Space: "foo",
				Key:   "bar",
			},
			space:     "foo",
			blockTime: 1,
			sender:    sender,
			expected: []*KeyValueMeta{
				{
					Key: "bar2",
					ValueMeta: &ValueMeta{
						Size:    6,
						Created: 2,
						Updated: 2,
					},
				},
			},
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
		if err := tv.utx.Execute(tc); err != nil {
			t.Fatalf("#%d: tx.Execute err expected nil, got %v", i, err)
		}

		s, exists, err := GetSpaceInfo(db, []byte(tv.space))
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatal("foo should exist")
		}

		kvs, err := GetAllValueMetas(db, s.RawSpace)
		if err != nil {
			t.Fatal(err)
		}

		for i, ex := range tv.expected {
			corr := kvs[i]
			if ex.Key != corr.Key {
				t.Fatalf("%d: keys not equal expected=%+v observed=%+v", i, ex.Key, corr.Key)
			}
			if ex.ValueMeta.Created != corr.ValueMeta.Created {
				t.Fatalf("%d: created not equal expected=%+v observed=%+v", i, ex.ValueMeta.Created, corr.ValueMeta.Created)
			}
			if ex.ValueMeta.Updated != corr.ValueMeta.Updated {
				t.Fatalf("%d: updated not equal expected=%+v observed=%+v", i, ex.ValueMeta.Updated, corr.ValueMeta.Updated)
			}
			if ex.ValueMeta.Size != corr.ValueMeta.Size {
				t.Fatalf("%d: size not equal expected=%+v observed=%+v", i, ex.ValueMeta.Size, corr.ValueMeta.Size)
			}
		}
	}
}
