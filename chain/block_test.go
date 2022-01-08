// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"errors"
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
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
							Hght:   1, Difficulty: 1, Cost: 1,
						},
						st: choices.Processing,
					},
					2,
					&Context{NextDifficulty: 1, NextCost: 1},
					nil,
					0,
					nil,
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
							Hght:   1, Difficulty: 1, Cost: 1,
						},
						st: choices.Processing,
					},
					1,
					&Context{NextDifficulty: 1, NextCost: 1},
					nil,
					1,
					nil,
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
							Hght:   1, Difficulty: 1, Cost: 1,
						},
						st: choices.Processing,
					},
					int64(futureBound+time.Hour),
					&Context{NextDifficulty: 1, NextCost: 1},
					nil,
					1,
					nil,
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
							Hght:   1, Difficulty: 1, Cost: 1,
						},
						st: choices.Processing,
					},
					2,
					&Context{NextDifficulty: 1, NextCost: 1},
					&Context{NextDifficulty: 1, NextCost: 1},
					1,
					nil,
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
							Hght:   1, Difficulty: 1000, Cost: 1000,
						},
						st: choices.Accepted,
					},
					2,
					&Context{NextDifficulty: 1, NextCost: 1},
					&Context{NextDifficulty: 1, NextCost: 1000},
					1,
					nil,
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
							Hght:   1, Difficulty: 1000, Cost: 1000,
						},
						st: choices.Accepted,
					},
					2,
					&Context{NextDifficulty: 1, NextCost: 1},
					&Context{NextDifficulty: 1000, NextCost: 1},
					1,
					nil,
				)
				return blk
			},
			expectedVerifyErr: ErrInvalidDifficulty,
		},
		{
			createBlk: func() *StatelessBlock {
				blk := createTestBlk(
					t,
					&StatelessBlock{
						StatefulBlock: &StatefulBlock{
							Tmstmp: 1,
							Prnt:   ids.GenerateTestID(),
							Hght:   1, Difficulty: 1, Cost: 1,
						},
						st: choices.Accepted,
					},
					2,
					&Context{NextDifficulty: 1, NextCost: 1000},
					nil,
					1,
					[]byte("a"),
				)
				return blk
			},
			expectedVerifyErr: ErrInvalidExtraData,
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
	extraData []byte,
) *StatelessBlock {
	t.Helper()

	ctrl := gomock.NewController(t)
	vm := NewMockVM(ctrl)
	parentBlk.vm = vm

	if err := parentBlk.init(); err != nil {
		t.Fatal(err)
	}
	vm.EXPECT().GetBlock(parentBlk.ID()).Return(parentBlk, nil)

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
	for i := 0; i < txsN; i++ {
		blk.StatefulBlock.Txs[i] = createTestClaimTx(t, blk.id, 100)
	}
	blk.StatefulBlock.ExtraData = extraData
	if execCtx != nil {
		execCtx.RecentBlockIDs.Add(parentBlk.ID(), blk.id)
		vm.EXPECT().ExecutionContext(blkTmpstp, parentBlk).Return(execCtx, nil)
	}

	return blk
}
