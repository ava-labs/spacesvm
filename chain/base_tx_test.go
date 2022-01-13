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

	tt := []struct {
		tx  *BaseTx
		err error
	}{
		{
			tx:  &BaseTx{Pfx: []byte("foo"), BlkID: ids.GenerateTestID()},
			err: nil,
		},
		{
			tx:  &BaseTx{Pfx: []byte("fo/a")},
			err: parser.ErrInvalidDelimiter,
		},
		{
			tx:  &BaseTx{Pfx: []byte("foo/")},
			err: parser.ErrInvalidDelimiter,
		},
		{
			tx:  &BaseTx{Pfx: []byte("foo")},
			err: ErrInvalidBlockID,
		},
		{
			tx: &BaseTx{
				BlkID: ids.GenerateTestID(),
				Pfx:   nil,
			},
			err: parser.ErrPrefixEmpty,
		},
		{
			tx: &BaseTx{
				BlkID: ids.GenerateTestID(),
				Pfx:   bytes.Repeat([]byte{'a'}, parser.MaxPrefixSize+1),
			},
			err: parser.ErrPrefixTooBig,
		},
	}
	g := DefaultGenesis()
	for i, tv := range tt {
		err := tv.tx.ExecuteBase(g)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.err, err)
		}
	}
}
