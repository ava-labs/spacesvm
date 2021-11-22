// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/quarkvm/crypto/ed25519"
)

func TestPut(t *testing.T) {
	s := New(snow.DefaultContextTest(), memdb.New())
	defer s.Close()

	priv, err := ed25519.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.PublicKey()

	tt := []struct {
		k    []byte
		v    []byte
		opts []OpOption
		err  error
	}{
		{
			k:    []byte("foo"),
			v:    []byte("bar"),
			opts: []OpOption{WithPublicKey(pub)},
			err:  nil,
		},
		{
			k:    []byte("foo"),
			v:    []byte("bar"),
			opts: []OpOption{WithPublicKey(pub)},
			err:  ErrKeyExists,
		},
		{
			k:    []byte(""),
			v:    []byte("bar"),
			opts: nil,
			err:  ErrInvalidKeyLength,
		},
		{
			k:    bytes.Repeat([]byte(" "), maxKeyLength+1),
			v:    []byte("bar"),
			opts: nil,
			err:  ErrInvalidKeyLength,
		},
		{
			k:    []byte("foo"),
			v:    bytes.Repeat([]byte(" "), maxValueLength+1),
			opts: nil,
			err:  ErrInvalidValueLength,
		},
	}
	for i, tv := range tt {
		err := s.Put(tv.k, tv.v, tv.opts...)
		if err != tv.err {
			t.Fatalf("#%d: put err expected %v, got %v", i, tv.err, err)
		}
	}
}

func TestRange(t *testing.T) {
	s := New(snow.DefaultContextTest(), memdb.New())
	defer s.Close()

	priv1, err := ed25519.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pub1 := priv1.PublicKey()

	priv2, err := ed25519.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pub2 := priv2.PublicKey()

	for i := 0; i < 5; i++ {
		if err = s.Put(
			[]byte(fmt.Sprintf("foo/%d", i)),
			[]byte(fmt.Sprintf("bar%05d", i)),
			WithPublicKey(pub1),
		); err != nil {
			t.Fatal(err)
		}
	}

	tt := []struct {
		k    []byte
		opts []OpOption
		resp RangeResponse
		err  error
	}{
		{ // prefix exists but the key itself does not exist
			k:    []byte("foo/9"),
			opts: []OpOption{WithPublicKey(pub1)},
			resp: RangeResponse{},
			err:  nil,
		},
		{ // invalid pub key
			k:    []byte("foo"),
			opts: []OpOption{WithPublicKey(pub2)},
			resp: RangeResponse{},
			err:  ErrPubKeyNotAllowed,
		},
		{ // key itself does not exist
			k:    []byte("not-exist"),
			opts: []OpOption{WithPublicKey(pub1)},
			resp: RangeResponse{},
			err:  ErrKeyNotExist,
		},
		{ // single key
			k:    []byte("foo/1"),
			opts: []OpOption{WithPublicKey(pub1)},
			resp: RangeResponse{
				KeyValues: []KeyValue{
					{Key: []byte("foo/1"), Value: []byte("bar00001")},
				},
			},
			err: nil,
		},
		{ // prefix query
			k:    []byte("foo"),
			opts: []OpOption{WithPublicKey(pub1), WithPrefix()},
			resp: RangeResponse{
				KeyValues: []KeyValue{
					{Key: []byte("foo/0"), Value: []byte("bar00000")},
					{Key: []byte("foo/1"), Value: []byte("bar00001")},
					{Key: []byte("foo/2"), Value: []byte("bar00002")},
					{Key: []byte("foo/3"), Value: []byte("bar00003")},
					{Key: []byte("foo/4"), Value: []byte("bar00004")},
				},
			},
			err: nil,
		},
		{ // range query
			k:    []byte("foo"),
			opts: []OpOption{WithPublicKey(pub1), WithRangeEnd("foo/3")},
			resp: RangeResponse{
				KeyValues: []KeyValue{
					{Key: []byte("foo/0"), Value: []byte("bar00000")},
					{Key: []byte("foo/1"), Value: []byte("bar00001")},
					{Key: []byte("foo/2"), Value: []byte("bar00002")},
				},
			},
			err: nil,
		},
		{ // range query
			k:    []byte("foo/2"),
			opts: []OpOption{WithPublicKey(pub1), WithRangeEnd("foo/5")},
			resp: RangeResponse{
				KeyValues: []KeyValue{
					{Key: []byte("foo/2"), Value: []byte("bar00002")},
					{Key: []byte("foo/3"), Value: []byte("bar00003")},
					{Key: []byte("foo/4"), Value: []byte("bar00004")},
				},
			},
			err: nil,
		},
	}
	for i, tv := range tt {
		resp, err := s.Range(tv.k, tv.opts...)
		if err != tv.err {
			t.Fatalf("#%d: range err expected %v, got %v", i, tv.err, err)
		}
		if tv.err != nil {
			continue
		}
		rv := *resp
		if !reflect.DeepEqual(rv, tv.resp) {
			t.Fatalf("#%d: range response expected %v, got %v", i, tv.resp, rv)
		}
	}
}
