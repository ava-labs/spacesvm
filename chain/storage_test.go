// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/quarkvm/parser"
)

func TestPrefixValueKey(t *testing.T) {
	t.Parallel()

	tt := []struct {
		pfx      []byte
		key      []byte
		valueKey []byte
	}{
		{
			pfx:      []byte("foo"),
			key:      []byte("hello"),
			valueKey: append([]byte{keyPrefix}, []byte("/foo/hello")...),
		},
		{
			pfx:      []byte("foo/"),
			key:      []byte("hello"),
			valueKey: append([]byte{keyPrefix}, []byte("/foo/hello")...),
		},
	}
	for i, tv := range tt {
		vv := PrefixValueKey(tv.pfx, tv.key)
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

func TestRange(t *testing.T) {
	t.Parallel()

	db := memdb.New()
	defer db.Close()

	for i := 0; i < 5; i++ {
		if err := PutPrefixKey(
			db,
			[]byte("foo"),
			[]byte(fmt.Sprintf("hello%05d", i)),
			[]byte(fmt.Sprintf("bar%05d", i)),
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
			pfx:  []byte("foo/"),
			key:  []byte("9"),
			opts: nil,
			kvs:  nil,
		},
		{ // single key
			pfx:  []byte("foo/"),
			key:  []byte("hello00000"),
			opts: nil,
			kvs: []KeyValue{
				{Key: []byte("foo/hello00000"), Value: []byte("bar00000")},
			},
		},
		{ // prefix query
			pfx:  []byte("foo/"),
			key:  []byte("hello"),
			opts: []OpOption{WithPrefix()},
			kvs: []KeyValue{
				{Key: []byte("foo/hello00000"), Value: []byte("bar00000")},
				{Key: []byte("foo/hello00001"), Value: []byte("bar00001")},
				{Key: []byte("foo/hello00002"), Value: []byte("bar00002")},
				{Key: []byte("foo/hello00003"), Value: []byte("bar00003")},
				{Key: []byte("foo/hello00004"), Value: []byte("bar00004")},
			},
		},
		{ // prefix query
			pfx:  []byte("foo/"),
			key:  nil,
			opts: []OpOption{WithPrefix()},
			kvs: []KeyValue{
				{Key: []byte("foo/hello00000"), Value: []byte("bar00000")},
				{Key: []byte("foo/hello00001"), Value: []byte("bar00001")},
				{Key: []byte("foo/hello00002"), Value: []byte("bar00002")},
				{Key: []byte("foo/hello00003"), Value: []byte("bar00003")},
				{Key: []byte("foo/hello00004"), Value: []byte("bar00004")},
			},
		},
		{ // prefix query
			pfx:  []byte("foo/"),
			key:  []byte("x"),
			opts: []OpOption{WithPrefix()},
			kvs:  nil,
		},
		{ // range query
			pfx:  []byte("foo/"),
			key:  []byte("hello"),
			opts: []OpOption{WithRangeEnd([]byte("hello00003"))},
			kvs: []KeyValue{
				{Key: []byte("foo/hello00000"), Value: []byte("bar00000")},
				{Key: []byte("foo/hello00001"), Value: []byte("bar00001")},
				{Key: []byte("foo/hello00002"), Value: []byte("bar00002")},
			},
		},
		{ // range query
			pfx:  []byte("foo/"),
			key:  []byte("hello00001"),
			opts: []OpOption{WithRangeEnd([]byte("hello00003"))},
			kvs: []KeyValue{
				{Key: []byte("foo/hello00001"), Value: []byte("bar00001")},
				{Key: []byte("foo/hello00002"), Value: []byte("bar00002")},
			},
		},
		{ // range query
			pfx:  []byte("foo/"),
			key:  []byte("hello00003"),
			opts: []OpOption{WithRangeEnd([]byte("hello00005"))},
			kvs: []KeyValue{
				{Key: []byte("foo/hello00003"), Value: []byte("bar00003")},
				{Key: []byte("foo/hello00004"), Value: []byte("bar00004")},
			},
		},
		{ // range query with limit
			pfx:  []byte("foo/"),
			key:  []byte("hello00003"),
			opts: []OpOption{WithRangeEnd([]byte("hello00005")), WithRangeLimit(1)},
			kvs: []KeyValue{
				{Key: []byte("foo/hello00003"), Value: []byte("bar00003")},
			},
		},
		{ // prefix query with limit
			pfx:  []byte("foo/"),
			key:  []byte("hello"),
			opts: []OpOption{WithPrefix(), WithRangeLimit(3)},
			kvs: []KeyValue{
				{Key: []byte("foo/hello00000"), Value: []byte("bar00000")},
				{Key: []byte("foo/hello00001"), Value: []byte("bar00001")},
				{Key: []byte("foo/hello00002"), Value: []byte("bar00002")},
			},
		},
	}
	for i, tv := range tt {
		kvs := Range(db, tv.pfx, tv.key, tv.opts...)
		if len(tv.kvs) == 0 && len(kvs) == 0 {
			continue
		}
		if !reflect.DeepEqual(kvs, tv.kvs) {
			t.Fatalf("#%d: range response expected %d pair(s), got %v pair(s)", i, len(tv.kvs), len(kvs))
		}
	}
}
