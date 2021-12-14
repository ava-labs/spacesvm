// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/quarkvm/crypto"
)

func TestBaseTx(t *testing.T) {
	t.Parallel()

	priv, err := crypto.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.PublicKey()

	tt := []struct {
		tx  *BaseTx
		err error
	}{
		{
			tx:  &BaseTx{Sender: pub.Bytes(), Prefix: []byte("foo"), BlockID: ids.GenerateTestID()},
			err: nil,
		},
		{
			tx:  &BaseTx{Prefix: []byte("foo"), BlockID: ids.GenerateTestID()},
			err: ErrInvalidSender,
		},
		{
			tx:  &BaseTx{Sender: pub.Bytes(), Prefix: []byte("foo/")},
			err: ErrPrefixHasDelimiter,
		},
		{
			tx:  &BaseTx{Sender: pub.Bytes(), Prefix: []byte("foo")},
			err: ErrInvalidBlockID,
		},
		{
			tx:  &BaseTx{Sender: pub.Bytes(), Prefix: nil},
			err: ErrPrefixEmpty,
		},
		{
			tx:  &BaseTx{Sender: pub.Bytes(), Prefix: bytes.Repeat([]byte{'a'}, MaxPrefixSize+1)},
			err: ErrPrefixTooBig,
		},
	}
	for i, tv := range tt {
		err := tv.tx.ExecuteBase()
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.err, err)
		}
	}
}
