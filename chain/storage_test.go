// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"errors"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/spacesvm/parser"
)

func TestPrefixValueKey(t *testing.T) {
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
		vv := PrefixValueKey(tv.rpfx, tv.key)
		if !bytes.Equal(tv.valueKey, vv) {
			t.Fatalf("#%d: value expected %q, got %q", i, tv.valueKey, vv)
		}
	}
}

func TestPrefixInfoKey(t *testing.T) {
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
		vv := PrefixInfoKey(tv.pfx)
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
			txKey: append([]byte{txPrefix, parser.Delimiter}, id[:]...),
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
			blockKey: append([]byte{blockPrefix, parser.Delimiter}, id[:]...),
		},
	}
	for i, tv := range tt {
		vv := PrefixBlockKey(tv.blkID)
		if !bytes.Equal(tv.blockKey, vv) {
			t.Fatalf("#%d: value expected %q, got %q", i, tv.blockKey, vv)
		}
	}
}

func TestPutPrefixInfoAndKey(t *testing.T) {
	t.Parallel()

	db := memdb.New()
	defer db.Close()

	pfx := []byte("foo")
	k, v := []byte("k"), []byte("v")

	// expect failures for non-existing prefixInfo
	if ok, err := HasPrefix(db, pfx); ok || err != nil {
		t.Fatalf("unexpected ok %v, err %v", ok, err)
	}
	if ok, err := HasPrefixKey(db, pfx, k); ok || err != nil {
		t.Fatalf("unexpected ok %v, err %v", ok, err)
	}
	if err := PutPrefixKey(db, pfx, k, v); !errors.Is(err, ErrPrefixMissing) {
		t.Fatalf("unexpected error %v, expected %v", err, ErrPrefixMissing)
	}

	if err := PutPrefixInfo(
		db,
		pfx,
		&PrefixInfo{
			RawPrefix: ids.ShortID{0x1},
		},
		0,
	); err != nil {
		t.Fatal(err)
	}
	if err := PutPrefixKey(db, pfx, k, v); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	// expect success for existing prefixInfo
	if ok, err := HasPrefix(db, pfx); !ok || err != nil {
		t.Fatalf("unexpected ok %v, err %v", ok, err)
	}
	if ok, err := HasPrefixKey(db, pfx, k); !ok || err != nil {
		t.Fatalf("unexpected ok %v, err %v", ok, err)
	}
}

func TestRange(t *testing.T) {
	t.Parallel()

	db := memdb.New()
	defer db.Close()

	// Persist PrefixInfo so keys can be stored under rprefix
	pfx := []byte("foo")
	if err := PutPrefixInfo(db, pfx, &PrefixInfo{
		RawPrefix: ids.ShortID{0x1},
	}, 0); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		id := ids.GenerateTestID()
		if err := db.Put(PrefixTxValueKey(id), []byte(fmt.Sprintf("bar%05d", i))); err != nil {
			t.Fatal(err)
		}
		if err := PutPrefixKey(
			db,
			pfx,
			[]byte(fmt.Sprintf("hello%05d", i)),
			id[:],
		); err != nil {
			t.Fatal(err)
		}
	}

	tt := []struct {
		pfx  []byte
		key  []byte
		opts []OpOption
		kvs  []KeyValue
	}{
		{ // prefix exists but the key itself does not exist
			pfx:  pfx,
			key:  []byte("9"),
			opts: nil,
			kvs:  nil,
		},
		{ // single key
			pfx:  pfx,
			key:  []byte("hello00000"),
			opts: nil,
			kvs: []KeyValue{
				{Key: []byte("hello00000"), Value: []byte("bar00000")},
			},
		},
		{ // prefix query
			pfx:  pfx,
			key:  []byte("hello"),
			opts: []OpOption{WithPrefix()},
			kvs: []KeyValue{
				{Key: []byte("hello00000"), Value: []byte("bar00000")},
				{Key: []byte("hello00001"), Value: []byte("bar00001")},
				{Key: []byte("hello00002"), Value: []byte("bar00002")},
				{Key: []byte("hello00003"), Value: []byte("bar00003")},
				{Key: []byte("hello00004"), Value: []byte("bar00004")},
			},
		},
		{ // prefix query
			pfx:  pfx,
			key:  nil,
			opts: []OpOption{WithPrefix()},
			kvs: []KeyValue{
				{Key: []byte("hello00000"), Value: []byte("bar00000")},
				{Key: []byte("hello00001"), Value: []byte("bar00001")},
				{Key: []byte("hello00002"), Value: []byte("bar00002")},
				{Key: []byte("hello00003"), Value: []byte("bar00003")},
				{Key: []byte("hello00004"), Value: []byte("bar00004")},
			},
		},
		{ // prefix query
			pfx:  pfx,
			key:  []byte("x"),
			opts: []OpOption{WithPrefix()},
			kvs:  nil,
		},
		{ // range query
			pfx:  pfx,
			key:  []byte("hello"),
			opts: []OpOption{WithRangeEnd([]byte("hello00003"))},
			kvs: []KeyValue{
				{Key: []byte("hello00000"), Value: []byte("bar00000")},
				{Key: []byte("hello00001"), Value: []byte("bar00001")},
				{Key: []byte("hello00002"), Value: []byte("bar00002")},
			},
		},
		{ // range query
			pfx:  pfx,
			key:  []byte("hello00001"),
			opts: []OpOption{WithRangeEnd([]byte("hello00003"))},
			kvs: []KeyValue{
				{Key: []byte("hello00001"), Value: []byte("bar00001")},
				{Key: []byte("hello00002"), Value: []byte("bar00002")},
			},
		},
		{ // range query
			pfx:  pfx,
			key:  []byte("hello00003"),
			opts: []OpOption{WithRangeEnd([]byte("hello00005"))},
			kvs: []KeyValue{
				{Key: []byte("hello00003"), Value: []byte("bar00003")},
				{Key: []byte("hello00004"), Value: []byte("bar00004")},
			},
		},
		{ // range query with limit
			pfx:  pfx,
			key:  []byte("hello00003"),
			opts: []OpOption{WithRangeEnd([]byte("hello00005")), WithRangeLimit(1)},
			kvs: []KeyValue{
				{Key: []byte("hello00003"), Value: []byte("bar00003")},
			},
		},
		{ // prefix query with limit
			pfx:  pfx,
			key:  []byte("hello"),
			opts: []OpOption{WithPrefix(), WithRangeLimit(3)},
			kvs: []KeyValue{
				{Key: []byte("hello00000"), Value: []byte("bar00000")},
				{Key: []byte("hello00001"), Value: []byte("bar00001")},
				{Key: []byte("hello00002"), Value: []byte("bar00002")},
			},
		},
	}
	for i, tv := range tt {
		kvs, err := Range(db, tv.pfx, tv.key, tv.opts...)
		if err != nil {
			t.Fatalf("#%d: unexpected error when fetching range %v", i, err)
		}
		if len(tv.kvs) == 0 && len(kvs) == 0 {
			continue
		}
		if !reflect.DeepEqual(kvs, tv.kvs) {
			t.Fatalf("#%d: range response expected %v pair(s), got %v pair(s)", i, tv.kvs, kvs)
		}
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
