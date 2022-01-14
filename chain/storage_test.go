// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"errors"
	"reflect"
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
		rpfx     ids.ShortID
		key      []byte
		valueKey []byte
	}{
		{
			rpfx:     ids.ShortID{0x1},
			key:      []byte("hello"),
			valueKey: append([]byte{keyPrefix}, []byte("/\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00/hello")...), //nolint:lll
		},
	}
	for i, tv := range tt {
		vv := SpaceValueKey(tv.rpfx, tv.key)
		if !bytes.Equal(tv.valueKey, vv) {
			t.Fatalf("#%d: value expected %q, got %q", i, tv.valueKey, vv)
		}
	}
}

func TestSpaceInfoKey(t *testing.T) {
	t.Parallel()

	tt := []struct {
		pfx     []byte
		infoKey []byte
	}{
		{
			pfx:     []byte("foo"),
			infoKey: append([]byte{infoPrefix}, []byte("/foo")...),
		},
	}
	for i, tv := range tt {
		vv := SpaceInfoKey(tv.pfx)
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

	pfx := []byte("foo")
	k, v := []byte("k"), []byte("v")

	// expect failures for non-existing prefixInfo
	if ok, err := HasSpace(db, pfx); ok || err != nil {
		t.Fatalf("unexpected ok %v, err %v", ok, err)
	}
	if ok, err := HasSpaceKey(db, pfx, k); ok || err != nil {
		t.Fatalf("unexpected ok %v, err %v", ok, err)
	}
	if err := PutSpaceKey(db, pfx, k, v); !errors.Is(err, ErrSpaceMissing) {
		t.Fatalf("unexpected error %v, expected %v", err, ErrSpaceMissing)
	}

	if err := PutSpaceInfo(
		db,
		pfx,
		&SpaceInfo{
			RawSpace: ids.ShortID{0x1},
		},
		0,
	); err != nil {
		t.Fatal(err)
	}
	if err := PutSpaceKey(db, pfx, k, v); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	// expect success for existing prefixInfo
	if ok, err := HasSpace(db, pfx); !ok || err != nil {
		t.Fatalf("unexpected ok %v, err %v", ok, err)
	}
	if ok, err := HasSpaceKey(db, pfx, k); !ok || err != nil {
		t.Fatalf("unexpected ok %v, err %v", ok, err)
	}
}

func TestSpecificTimeKey(t *testing.T) {
	rpfx0 := ids.ShortID{'k'}
	k := PrefixExpiryKey(100, rpfx0)
	ts, rpfx, err := extractSpecificTimeKey(k)
	if err != nil {
		t.Fatal(err)
	}
	if ts != 100 {
		t.Fatalf("unexpected timestamp %d, expected 100", ts)
	}
	if rpfx != rpfx0 {
		t.Fatalf("unexpected rawPrefix %v, expected %v", rpfx, rpfx0)
	}

	if _, _, err = extractSpecificTimeKey(k[:10]); !errors.Is(err, ErrInvalidKeyFormat) {
		t.Fatalf("unexpected error %v, expected %v", err, ErrInvalidKeyFormat)
	}
}

func TestGetAllValues(t *testing.T) {
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
		expected  []*KeyValue
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
			expected:  []*KeyValue{},
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
			expected: []*KeyValue{
				{Key: "bar", Value: []byte("value")},
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
			expected:  []*KeyValue{},
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
			expected: []*KeyValue{
				{Key: "bar", Value: []byte("value2")},
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
			blockTime: 1,
			sender:    sender,
			expected: []*KeyValue{
				{Key: "bar", Value: []byte("value2")},
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
			blockTime: 1,
			sender:    sender,
			expected: []*KeyValue{
				{Key: "bar", Value: []byte("value2")},
				{Key: "bar2", Value: []byte("value2")},
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
			expected: []*KeyValue{
				{Key: "bar2", Value: []byte("value2")},
			},
		},
	}
	for i, tv := range tt {
		if i > 0 {
			// Expire old prefixes between txs
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

		kvs, err := GetAllValues(db, s.RawSpace)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(tv.expected, kvs) {
			t.Fatalf("%d: values not equal expected=%+v observed=%+v", i, tv.expected, kvs)
		}
	}
}
