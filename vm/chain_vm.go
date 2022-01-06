// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/quarkvm/chain"
)

func (vm *VM) State() database.Database {
	return vm.db
}

func (vm *VM) Mempool() chain.Mempool {
	return vm.mempool
}

func (vm *VM) Verified(b *chain.StatelessBlock) {
	vm.verifiedBlocks[b.ID()] = b
	for _, tx := range b.Txs {
		_ = vm.mempool.Remove(tx.ID())
	}
	log.Debug("verified block", "id", b.ID(), "parent", b.Prnt)
}

func (vm *VM) Rejected(b *chain.StatelessBlock) {
	delete(vm.verifiedBlocks, b.ID())
	for _, tx := range b.Txs {
		vm.mempool.Add(tx)
	}
	log.Debug("rejected block", "id", b.ID())
}

func (vm *VM) Accepted(b *chain.StatelessBlock) {
	vm.blocks.Put(b.ID(), b)
	delete(vm.verifiedBlocks, b.ID())
	vm.lastAccepted = b
	log.Debug("accepted block", "id", b.ID())
}

func (vm *VM) ExecutionContext(currTime int64, lastBlock *chain.StatelessBlock) (*chain.Context, error) {
	recentBlockIDs := ids.Set{}
	recentTxIDs := ids.Set{}
	err := vm.lookback(currTime, lastBlock.ID(), func(b *chain.StatelessBlock) (bool, error) {
		recentBlockIDs.Add(b.ID())
		for _, tx := range b.StatefulBlock.Txs {
			recentTxIDs.Add(tx.ID())
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	// compute new block cost
	secondsSinceLast := currTime - lastBlock.Tmstmp
	nextCost := lastBlock.Cost
	if secondsSinceLast < chain.BlockTarget {
		nextCost += uint64(chain.BlockTarget - secondsSinceLast)
	} else {
		possibleDiff := uint64(secondsSinceLast - chain.BlockTarget)
		// TODO: clean this up
		if nextCost >= vm.minBlockCost && possibleDiff < nextCost-vm.minBlockCost {
			nextCost -= possibleDiff
		} else {
			nextCost = vm.minBlockCost
		}
	}

	// compute new min difficulty
	nextDifficulty := lastBlock.Difficulty
	if recentTxs := recentTxIDs.Len(); recentTxs > chain.TargetTransactions {
		nextDifficulty++
	} else if recentTxs < chain.TargetTransactions {
		elapsedWindows := uint64(secondsSinceLast/chain.LookbackWindow) + 1 // account for current window being less
		if nextDifficulty >= vm.minDifficulty && elapsedWindows < nextDifficulty-vm.minDifficulty {
			nextDifficulty -= elapsedWindows
		} else {
			nextDifficulty = vm.minDifficulty
		}
	}

	return &chain.Context{
		RecentBlockIDs: recentBlockIDs,
		RecentTxIDs:    recentTxIDs,
		NextCost:       nextCost,
		NextDifficulty: nextDifficulty,
	}, nil
}
