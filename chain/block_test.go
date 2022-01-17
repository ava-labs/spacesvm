// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"errors"
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ethereum/go-ethereum/crypto"
	gomock "github.com/golang/mock/gomock"
)

func TestBlock(t *testing.T) {
	t.Parallel()

	tt := []struct {
		createBlk         func() *StatelessBlock
		expectedVerifyErr error
	}{
		{
			createBlk: func() *StatelessBlock {
				blk := createTestBlk(
					t,
					&StatelessBlock{
						StatefulBlock: &StatefulBlock{
							Tmstmp: 1,
							Prnt:   ids.GenerateTestID(),
							Hght:   1, Price: 1, Cost: 1,
						},
						st: choices.Processing,
					},
					2,
					&Context{NextPrice: 1, NextCost: 1},
					nil,
					0,
				)
				return blk
			},
			expectedVerifyErr: ErrNoTxs,
		},
		{
			createBlk: func() *StatelessBlock {
				blk := createTestBlk(
					t,
					&StatelessBlock{
						StatefulBlock: &StatefulBlock{
							Tmstmp: 10,
							Prnt:   ids.GenerateTestID(),
							Hght:   1, Price: 1, Cost: 1,
						},
						st: choices.Processing,
					},
					1,
					&Context{NextPrice: 1, NextCost: 1},
					nil,
					1,
				)
				return blk
			},
			expectedVerifyErr: ErrTimestampTooEarly,
		},
		{
			createBlk: func() *StatelessBlock {
				blk := createTestBlk(
					t,
					&StatelessBlock{
						StatefulBlock: &StatefulBlock{
							Tmstmp: 1,
							Prnt:   ids.GenerateTestID(),
							Hght:   1, Price: 1, Cost: 1,
						},
						st: choices.Processing,
					},
					int64(futureBound+time.Hour),
					&Context{NextPrice: 1, NextCost: 1},
					nil,
					1,
				)
				return blk
			},
			expectedVerifyErr: ErrTimestampTooLate,
		},
		{
			createBlk: func() *StatelessBlock {
				blk := createTestBlk(
					t,
					&StatelessBlock{
						StatefulBlock: &StatefulBlock{
							Tmstmp: 1,
							Prnt:   ids.GenerateTestID(),
							Hght:   1, Price: 1, Cost: 1,
						},
						st: choices.Processing,
					},
					2,
					&Context{NextPrice: 1, NextCost: 1},
					&Context{NextPrice: 1, NextCost: 1},
					1,
				)
				return blk
			},
			expectedVerifyErr: ErrParentBlockNotVerified,
		},
		{
			createBlk: func() *StatelessBlock {
				blk := createTestBlk(
					t,
					&StatelessBlock{
						StatefulBlock: &StatefulBlock{
							Tmstmp: 1,
							Prnt:   ids.ID{0, 1, 2, 4, 5},
							Hght:   1, Price: 1000, Cost: 1000,
						},
						st: choices.Accepted,
					},
					2,
					&Context{NextPrice: 1, NextCost: 1},
					&Context{NextPrice: 1, NextCost: 1000},
					1,
				)
				return blk
			},
			expectedVerifyErr: ErrInvalidCost,
		},
		{
			createBlk: func() *StatelessBlock {
				blk := createTestBlk(
					t,
					&StatelessBlock{
						StatefulBlock: &StatefulBlock{
							Tmstmp: 1,
							Prnt:   ids.ID{0, 1, 2, 4, 5},
							Hght:   1, Price: 1000, Cost: 1000,
						},
						st: choices.Accepted,
					},
					2,
					&Context{NextPrice: 1, NextCost: 1},
					&Context{NextPrice: 1000, NextCost: 1},
					1,
				)
				return blk
			},
			expectedVerifyErr: ErrInvalidPrice,
		},
	}
	for i, tv := range tt {
		blk := tv.createBlk()
		err := blk.Verify()
		if !errors.Is(err, tv.expectedVerifyErr) {
			t.Fatalf("#%d: block verify expected error %v, got %v", i, tv.expectedVerifyErr, err)
		}
	}
}

func createTestBlk(
	t *testing.T,
	parentBlk *StatelessBlock,
	blkTmpstp int64,
	blkCtx *Context,
	execCtx *Context,
	txsN int,
) *StatelessBlock {
	t.Helper()

	ctrl := gomock.NewController(t)
	vm := NewMockVM(ctrl)
	vm.EXPECT().Genesis().Return(DefaultGenesis()).AnyTimes()
	parentBlk.vm = vm
	if err := parentBlk.init(); err != nil {
		t.Fatal(err)
	}
	vm.EXPECT().GetStatelessBlock(parentBlk.ID()).Return(parentBlk, nil).AnyTimes()

	blk := NewBlock(vm, parentBlk, blkTmpstp, blkCtx)
	if uint64(blk.StatefulBlock.Tmstmp) != uint64(blkTmpstp) {
		t.Fatalf("blk.StatefulBlock.Tmstmp expected %d, got %d", blkTmpstp, blk.StatefulBlock.Tmstmp)
	}
	if err := blk.init(); err != nil {
		t.Fatal(err)
	}
	if blk.id == ids.Empty {
		t.Fatal("unexpected empty ID after init")
	}
	blk.StatefulBlock.Txs = make([]*Transaction, txsN)
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < txsN; i++ {
		blk.StatefulBlock.Txs[i] = createTestTx(t, blk.id, priv)
	}
	if execCtx != nil {
		execCtx.RecentBlockIDs.Add(parentBlk.ID(), blk.id)
		vm.EXPECT().ExecutionContext(blkTmpstp, parentBlk).Return(execCtx, nil)
	}

	return blk
}
