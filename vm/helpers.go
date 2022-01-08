// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/quarkvm/chain"
)

// TODO: add caching + test
func (vm *VM) lookback(currTime int64, lastID ids.ID, f func(b *chain.StatelessBlock) (bool, error)) error {
	curr, err := vm.getBlock(lastID)
	if err != nil {
		return err
	}
	// Include at least parent block in the window, regardless of how old
	for curr != nil && (currTime-curr.Tmstmp <= chain.LookbackWindow || curr.ID() == lastID) {
		if cont, err := f(curr); !cont || err != nil {
			return err
		}
		if curr.Hght == 0 /* genesis */ {
			return nil
		}
		b, err := vm.getBlock(curr.Prnt)
		if err != nil {
			return err
		}
		curr = b
	}
	return nil
}

func (vm *VM) ValidBlockID(blockID ids.ID) (bool, error) {
	var foundBlockID bool
	err := vm.lookback(time.Now().Unix(), vm.preferred, func(b *chain.StatelessBlock) (bool, error) {
		if b.ID() == blockID {
			foundBlockID = true
			return false, nil
		}
		return true, nil
	})
	return foundBlockID, err
}

func (vm *VM) DifficultyEstimate() (uint64, uint64, error) {
	prnt, err := vm.GetBlock(vm.preferred)
	if err != nil {
		return 0, 0, err
	}
	parent, ok := prnt.(*chain.StatelessBlock)
	if !ok {
		return 0, 0, fmt.Errorf("unexpected snowman.Block %T, expected *StatelessBlock", prnt)
	}
	ctx, err := vm.ExecutionContext(time.Now().Unix(), parent)
	if err != nil {
		return 0, 0, err
	}
	recentTxs := ctx.RecentTxIDs.Len()
	recommendedC := vm.minBlockCost
	if recentTxs > 0 {
		recommendedC = ctx.NextCost / uint64(recentTxs)
	}
	return ctx.NextDifficulty, recommendedC, nil
}
