// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"errors"
	"testing"

	"github.com/ava-labs/avalanchego/ids"
)

func TestBaseTx(t *testing.T) {
	t.Parallel()

	tt := []struct {
		tx  *BaseTx
		err error
	}{
		{
			tx: &BaseTx{BlockID: ids.GenerateTestID(), Price: 1},
		},
		{
			tx:  &BaseTx{BlockID: ids.GenerateTestID()},
			err: ErrInvalidPrice,
		},
		{
			tx:  &BaseTx{},
			err: ErrInvalidBlockID,
		},
	}
	g := DefaultGenesis()
	for i, tv := range tt {
		err := tv.tx.ExecuteBase(g)
		if tv.err != nil && !errors.Is(err, tv.err) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.err, err)
		}
	}
}
