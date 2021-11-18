// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"bytes"
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
	sig, err := priv.Sign([]byte("bar"))
	if err != nil {
		panic(err)
	}

	tt := []struct {
		k    []byte
		v    []byte
		opts []OpOption
		err  error
	}{
		{
			k:    []byte("foo"),
			v:    []byte("bar"),
			opts: []OpOption{WithSignature(pub, sig)},
			err:  nil,
		},
		{
			k:    []byte("foo"),
			v:    []byte("bar"),
			opts: []OpOption{WithSignature(pub, []byte("bar"))},
			err:  ErrInvalidSig,
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
