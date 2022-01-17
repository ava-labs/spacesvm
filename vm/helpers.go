// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"fmt"
	"sort"
	"time"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/spacesvm/chain"
)

const (
	feePercentile = 60
)

// TODO: add caching + test
func (vm *VM) lookback(currTime int64, lastID ids.ID, f func(b *chain.StatelessBlock) (bool, error)) error {
	curr, err := vm.GetStatelessBlock(lastID)
	if err != nil {
		return err
	}
	// Include at least parent block in the window, regardless of how old
	for curr != nil && (currTime-curr.Tmstmp <= vm.genesis.LookbackWindow || curr.ID() == lastID) {
		if cont, err := f(curr); !cont || err != nil {
			return err
		}
		if curr.Hght == 0 /* genesis */ {
			return nil
		}
		b, err := vm.GetStatelessBlock(curr.Prnt)
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

func (vm *VM) SuggestedFee() (uint64, uint64, error) {
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

	// Sort useful costs/prices
	sort.Slice(ctx.Prices, func(i, j int) bool { return ctx.Prices[i] < ctx.Prices[j] })
	pPrice := ctx.Prices[(len(ctx.Prices)-1)*feePercentile/100]
	if g := vm.genesis; pPrice < g.MinPrice {
		pPrice = g.MinPrice
	}
	sort.Slice(ctx.Costs, func(i, j int) bool { return ctx.Costs[i] < ctx.Costs[j] })
	pCost := ctx.Costs[(len(ctx.Costs)-1)*feePercentile/100]
	if pCost < chain.MinBlockCost {
		pCost = chain.MinBlockCost
	}

	// Adjust cost estimate based on recent txs
	recentTxs := ctx.RecentTxIDs.Len()
	if recentTxs == 0 {
		return pPrice, pCost, nil
	}
	cPerTx := pCost / uint64(recentTxs) / uint64(ctx.RecentBlockIDs.Len())
	if cPerTx < chain.MinBlockCost {
		// We always recommend at least the minBlockCost in case there are no other
		// transactions.
		cPerTx = chain.MinBlockCost
	}
	return pPrice, cPerTx, nil
}
