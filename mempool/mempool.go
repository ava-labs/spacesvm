// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mempool

import (
	"container/heap"
	"sync"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/quarkvm/chain"
)

var _ chain.Mempool = &Mempool{}

type Mempool struct {
	mu      sync.RWMutex
	g       *chain.Genesis
	maxSize int
	maxHeap *internalTxHeap
	minHeap *internalTxHeap

	// Pending is a channel of length one, which the mempool ensures has an item on
	// it as long as there is an unissued transaction remaining in [txs]
	Pending chan struct{}
	// newTxs is an array of [Tx] that are ready to be gossiped.
	newTxs []*chain.Transaction
}

// New creates a new [Mempool]. [maxSize] must be > 0 or else the
// implementation may panic.
func New(g *chain.Genesis, maxSize int) *Mempool {
	return &Mempool{
		g:       g,
		maxSize: maxSize,
		maxHeap: newInternalTxHeap(maxSize, false),
		minHeap: newInternalTxHeap(maxSize, true),
		Pending: make(chan struct{}, 1),
	}
}

func (th *Mempool) Add(tx *chain.Transaction) bool {
	txID := tx.ID()
	// Don't add duplicates
	if th.Has(txID) {
		return false
	}
	// Optimistically add tx to mempool
	difficulty := tx.Difficulty()
	oldLen := th.Len()

	th.mu.Lock()
	heap.Push(th.maxHeap, &txEntry{
		id:         txID,
		difficulty: difficulty,
		tx:         tx,
		index:      oldLen,
	})
	heap.Push(th.minHeap, &txEntry{
		id:         txID,
		difficulty: difficulty,
		tx:         tx,
		index:      oldLen,
	})
	th.mu.Unlock()

	// Remove the lowest paying tx
	//
	// Note: we do this after adding the new transaction in case it is the new
	// lowest paying transaction
	if th.Len() >= th.maxSize {
		t := th.PopMin()
		if t.ID() == txID {
			return false
		}
	}
	// When adding [tx] to the mempool make sure that there is an item in Pending
	// to signal the VM to produce a block. Note: if the VM's buildStatus has already
	// been set to something other than [dontBuild], this will be ignored and won't be
	// reset until the engine calls BuildBlock. This case is handled in IssueCurrentTx
	// and CancelCurrentTx.
	th.newTxs = append(th.newTxs, tx)
	th.addPending()
	return true
}

// Assumes there is non-zero items in [Mempool]
func (th *Mempool) PeekMax() (*chain.Transaction, uint64) {
	th.mu.RLock()
	txEntry := th.maxHeap.items[0]
	th.mu.RUnlock()
	return txEntry.tx, txEntry.difficulty
}

// Assumes there is non-zero items in [Mempool]
func (th *Mempool) PeekMin() (*chain.Transaction, uint64) {
	th.mu.RLock()
	txEntry := th.minHeap.items[0]
	th.mu.RUnlock()
	return txEntry.tx, txEntry.difficulty
}

// Assumes there is non-zero items in [Mempool]
func (th *Mempool) PopMax() (*chain.Transaction, uint64) {
	th.mu.RLock()
	item := th.maxHeap.items[0]
	th.mu.RUnlock()
	return th.Remove(item.id), item.difficulty
}

// Assumes there is non-zero items in [Mempool]
func (th *Mempool) PopMin() *chain.Transaction {
	return th.Remove(th.minHeap.items[0].id)
}

func (th *Mempool) Remove(id ids.ID) *chain.Transaction { // O(log N)
	th.mu.Lock()
	defer th.mu.Unlock()

	maxEntry, ok := th.maxHeap.Get(id) // O(1)
	if !ok {
		return nil
	}
	heap.Remove(th.maxHeap, maxEntry.index) // O(log N)

	minEntry, ok := th.minHeap.Get(id) // O(1)
	if !ok {
		// This should never happen, as that would mean the heaps are out of
		// sync.
		return nil
	}
	return heap.Remove(th.minHeap, minEntry.index).(*txEntry).tx // O(log N)
}

// Prune removes all transactions that are not found in "validHashes".
func (th *Mempool) Prune(validHashes ids.Set) {
	th.mu.RLock()
	toRemove := []ids.ID{}
	for _, txE := range th.maxHeap.items { // O(N)
		if !validHashes.Contains(txE.tx.GetBlockID()) {
			toRemove = append(toRemove, txE.id)
		}
	}
	th.mu.RUnlock()
	for _, txID := range toRemove { // O(K * log N)
		th.Remove(txID)
	}
}

func (th *Mempool) Len() int {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.maxHeap.Len()
}

func (th *Mempool) Get(id ids.ID) (*chain.Transaction, bool) {
	txEntry, ok := th.maxHeap.Get(id)
	if !ok {
		return nil, false
	}
	return txEntry.tx, true
}

func (th *Mempool) Has(id ids.ID) bool {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.maxHeap.Has(id)
}

// GetNewTxs returns the array of [newTxs] and replaces it with a new array.
func (th *Mempool) NewTxs(maxUnits uint64) []*chain.Transaction {
	th.mu.Lock()
	defer th.mu.Unlock()

	units := uint64(0)
	selected := []*chain.Transaction{}
	for i, tx := range th.newTxs {
		// It is possible that a block may have been accepted that contains some
		// new transactions before [NewTxs] is called.
		if !th.maxHeap.Has(tx.ID()) {
			continue
		}
		if tx.LoadUnits(th.g)+units > maxUnits {
			// Note: this algorithm preserves the ordering of new transactions
			th.newTxs = th.newTxs[i:]
			return selected
		}
		selected = append(selected, tx)
		units += tx.LoadUnits(th.g)
	}
	th.newTxs = nil
	return selected
}

// addPending makes sure that an item is in the Pending channel.
func (th *Mempool) addPending() {
	select {
	case th.Pending <- struct{}{}:
	default:
	}
}
