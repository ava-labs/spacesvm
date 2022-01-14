// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/spacesvm/chain"
)

func (vm *VM) Genesis() *chain.Genesis {
	return vm.genesis
}

func (vm *VM) IsBootstrapped() bool {
	return vm.bootstrapped
}

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
	log.Debug("accepted block", "blkID", b.ID(), "beneficiary", string(b.Beneficiary))
}

func (vm *VM) SetBeneficiary(prefix []byte) {
	vm.beneficiaryLock.Lock()
	defer vm.beneficiaryLock.Unlock()
	vm.beneficiary = prefix
}

func (vm *VM) Beneficiary() []byte {
	vm.beneficiaryLock.RLock()
	defer vm.beneficiaryLock.RUnlock()
	return vm.beneficiary
}

func (vm *VM) ExecutionContext(currTime int64, lastBlock *chain.StatelessBlock) (*chain.Context, error) {
	g := vm.genesis
	recentBlockIDs := ids.Set{}
	recentTxIDs := ids.Set{}
	recentUnits := uint64(0)
	prices := []uint64{}
	costs := []uint64{}
	err := vm.lookback(currTime, lastBlock.ID(), func(b *chain.StatelessBlock) (bool, error) {
		recentBlockIDs.Add(b.ID())
		for _, tx := range b.StatefulBlock.Txs {
			recentTxIDs.Add(tx.ID())
			recentUnits += tx.LoadUnits(g)
		}
		prices = append(prices, b.Price)
		costs = append(costs, b.Cost)
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	// compute new block cost
	secondsSinceLast := currTime - lastBlock.Tmstmp
	nextCost := lastBlock.Cost
	if secondsSinceLast < g.BlockTarget {
		nextCost += uint64(g.BlockTarget - secondsSinceLast)
	} else {
		possibleDiff := uint64(secondsSinceLast - g.BlockTarget)
		// TODO: clean this up
		if nextCost >= g.MinBlockCost && possibleDiff < nextCost-g.MinBlockCost {
			nextCost -= possibleDiff
		} else {
			nextCost = g.MinBlockCost
		}
	}

	// compute new min difficulty
	nextPrice := lastBlock.Price
	if recentUnits > g.TargetUnits {
		nextPrice++
	} else if recentUnits < g.TargetUnits {
		elapsedWindows := uint64(secondsSinceLast/g.LookbackWindow) + 1 // account for current window being less
		if nextPrice >= g.MinPrice && elapsedWindows < nextPrice-g.MinPrice {
			nextPrice -= elapsedWindows
		} else {
			nextPrice = g.MinPrice
		}
	}

	return &chain.Context{
		RecentBlockIDs:  recentBlockIDs,
		RecentTxIDs:     recentTxIDs,
		RecentLoadUnits: recentUnits,

		Prices: prices,
		Costs:  costs,

		NextPrice: nextPrice,
		NextCost:  nextCost,
	}, nil
}
