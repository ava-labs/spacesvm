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
		tx        *BaseTx
		baseTxErr error
		prefixErr error
	}{
		{
			tx: &BaseTx{Pfx: []byte("foo"), BlkID: ids.GenerateTestID(), Prce: 1},
		},
		{
			tx:        &BaseTx{Pfx: []byte("foo"), BlkID: ids.GenerateTestID()},
			baseTxErr: ErrInvalidPrice,
		},
		{
			tx:        &BaseTx{Pfx: []byte("fo/a"), BlkID: ids.GenerateTestID()},
			prefixErr: parser.ErrInvalidDelimiter,
		},
		{
			tx:        &BaseTx{Pfx: []byte("foo/"), BlkID: ids.GenerateTestID()},
			prefixErr: parser.ErrInvalidDelimiter,
		},
		{
			tx:        &BaseTx{Pfx: []byte("foo")},
			baseTxErr: ErrInvalidBlockID,
		},
		{
			tx: &BaseTx{
				BlkID: ids.GenerateTestID(),
				Pfx:   nil,
			},
			prefixErr: parser.ErrPrefixEmpty,
		},
		{
			tx: &BaseTx{
				BlkID: ids.GenerateTestID(),
				Pfx:   bytes.Repeat([]byte{'a'}, parser.MaxPrefixSize+1),
			},
			prefixErr: parser.ErrPrefixTooBig,
		},
	}
	g := DefaultGenesis()
	for i, tv := range tt {
		err := tv.tx.ExecuteBase(g)
		if tv.baseTxErr != nil && !errors.Is(err, tv.baseTxErr) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.baseTxErr, err)
		}
		if tv.prefixErr == nil {
			continue
		}
		tx := &ClaimTx{BaseTx: tv.tx}
		err = tx.Execute(&TransactionContext{Genesis: g})
		if !errors.Is(err, tv.prefixErr) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.prefixErr, err)
		}
	}
}
