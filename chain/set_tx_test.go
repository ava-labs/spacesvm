// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/quarkvm/crypto"
)

func TestSetTx(t *testing.T) {
	t.Parallel()

	priv, err := crypto.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.PublicKey()

	priv2, err := crypto.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pub2 := priv2.PublicKey()

	db := memdb.New()

	tt := []struct {
		utx       UnsignedTransaction
		blockTime int64
		err       error
	}{
		{ // successful claim
			utx: &ClaimTx{
				BaseTx: &BaseTx{
					Sender: pub.Bytes(),
					Prefix: []byte("foo/"),
				},
			},
			blockTime: 1,
			err:       nil,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					Sender:  pub.Bytes(),
					Prefix:  []byte("foo/"),
					BlockID: ids.GenerateTestID(),
				},
				Key:   []byte("bar"),
				Value: []byte("value"),
			},
			blockTime: 1,
			err:       nil,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					Sender:  pub.Bytes(),
					Prefix:  []byte("foo/"),
					BlockID: ids.GenerateTestID(),
				},
				Key: []byte("bar"),
			},
			blockTime: 1,
			err:       nil,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					Sender:  pub.Bytes(),
					Prefix:  []byte("foo/"),
					BlockID: ids.GenerateTestID(),
				},
				Key: []byte("bar"),
			},
			blockTime: 100,
			err:       ErrPrefixExpired,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					Sender:  pub2.Bytes(),
					Prefix:  []byte("foo/"),
					BlockID: ids.GenerateTestID(),
				},
				Key: []byte("bar"),
			},
			blockTime: 1,
			err:       ErrUnauthorized,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					Prefix:  []byte("foo/"),
					BlockID: ids.GenerateTestID(),
				},
			},
			blockTime: 1,
			err:       ErrKeyEmpty,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					Sender: pub.Bytes(),
					Prefix: []byte("foo/"),
				},
			},
			blockTime: 1,
			err:       ErrKeyEmpty,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					Sender: pub.Bytes(),
					Prefix: nil,
				},
			},
			blockTime: 1,
			err:       ErrPrefixEmpty,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					Sender: pub.Bytes(),
					Prefix: bytes.Repeat([]byte{'a'}, MaxPrefixSize+1),
				},
			},
			blockTime: 1,
			err:       ErrPrefixTooBig,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					Prefix:  []byte("foo/"),
					BlockID: ids.GenerateTestID(),
				},
				Key: bytes.Repeat([]byte{'a'}, MaxKeyLength+1),
			},
			blockTime: 1,
			err:       ErrKeyTooBig,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					Prefix:  []byte("foo/"),
					BlockID: ids.GenerateTestID(),
				},
				Key:   []byte("bar"),
				Value: bytes.Repeat([]byte{'b'}, MaxKeyLength+1),
			},
			blockTime: 1,
			err:       ErrValueTooBig,
		},
		{
			utx: &SetTx{
				BaseTx: &BaseTx{
					Sender:  pub.Bytes(),
					Prefix:  []byte("foo/"),
					BlockID: ids.GenerateTestID(),
				},
				Key: []byte("bar///"),
			},
			blockTime: 1,
			err:       ErrInvalidKeyDelimiter,
		},
	}
	for i, tv := range tt {
		err := tv.utx.Execute(db, tv.blockTime)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.err, err)
		}
	}
}
