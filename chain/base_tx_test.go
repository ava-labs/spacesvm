// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/quarkvm/parser"
)

func TestBaseTx(t *testing.T) {
	t.Parallel()

	priv, err := f.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	sender, err := FormatPK(priv.PublicKey())
	if err != nil {
		t.Fatal(err)
	}

	tt := []struct {
		tx  *BaseTx
		err error
	}{
		{
			tx:  &BaseTx{Sender: sender, Prefix: []byte("foo"), BlockID: ids.GenerateTestID()},
			err: nil,
		},
		{
			tx:  &BaseTx{Prefix: []byte("foo"), BlockID: ids.GenerateTestID()},
			err: ErrInvalidSender,
		},
		{
			tx:  &BaseTx{Sender: sender, Prefix: []byte("fo/a")},
			err: parser.ErrInvalidDelimiter,
		},
		{
			tx:  &BaseTx{Sender: sender, Prefix: []byte("foo/")},
			err: parser.ErrInvalidDelimiter,
		},
		{
			tx:  &BaseTx{Sender: sender, Prefix: []byte("foo")},
			err: ErrInvalidBlockID,
		},
		{
			tx: &BaseTx{
				Sender:  sender,
				BlockID: ids.GenerateTestID(),
				Prefix:  nil,
			},
			err: parser.ErrPrefixEmpty,
		},
		{
			tx: &BaseTx{
				Sender:  sender,
				BlockID: ids.GenerateTestID(),
				Prefix:  bytes.Repeat([]byte{'a'}, parser.MaxPrefixSize+1),
			},
			err: parser.ErrPrefixTooBig,
		},
	}
	for i, tv := range tt {
		err := tv.tx.ExecuteBase()
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.err, err)
		}
	}
}
