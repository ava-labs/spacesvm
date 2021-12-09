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
func (vm *VM) Verified(b *chain.Block) {
	vm.verifiedBlocks[b.ID()] = b
	for _, tx := range b.Txs {
		_ = vm.mempool.Remove(tx.ID())
	}
	log.Debug("verified block", "id", b.ID(), "parent", b.Prnt)
}
func (vm *VM) Rejected(b *chain.Block) {
	delete(vm.verifiedBlocks, b.ID())
	for _, tx := range b.Txs {
		vm.mempool.Add(tx)
	}
	log.Debug("rejected block", "id", b.ID())
}
func (vm *VM) Accepted(b *chain.Block) {
	delete(vm.verifiedBlocks, b.ID())
	vm.lastAccepted = b.ID()
	log.Debug("accepted block", "id", b.ID())
}
func (vm *VM) ExecutionContext(currTime int64, lastBlock *chain.Block) (*chain.Context, error) {
	recentBlockIDs := ids.Set{}
	recentTxIDs := ids.Set{}
	err := vm.lookback(currTime, lastBlock.ID(), func(b *chain.Block) (bool, error) {
		recentBlockIDs.Add(b.ID())
		for _, tx := range b.Txs {
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
		if possibleDiff < nextCost-chain.MinBlockCost {
			nextCost -= possibleDiff
		} else {
			nextCost = chain.MinBlockCost
		}
	}

	// compute new min difficulty
	nextDifficulty := lastBlock.Difficulty
	recentTxs := recentTxIDs.Len()
	if recentTxs > chain.TargetTransactions {
		nextDifficulty++
	} else if recentTxs < chain.TargetTransactions {
		elapsedWindows := uint64(secondsSinceLast/chain.LookbackWindow) + 1 // account for current window being less
		if elapsedWindows < nextDifficulty-chain.MinDifficulty {
			nextDifficulty -= elapsedWindows
		} else {
			nextDifficulty = chain.MinDifficulty
		}
	}

	return &chain.Context{RecentBlockIDs: recentBlockIDs, RecentTxIDs: recentTxIDs, NextCost: nextCost, NextDifficulty: nextDifficulty}, nil
}
