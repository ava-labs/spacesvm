// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package quarkvm

import (
	"container/heap"

	"github.com/ava-labs/avalanchego/ids"
)

// txEntry is used to track the work transactions pay to be included in
// the mempool.
type txEntry struct {
	id         ids.ID
	tx         transaction
	difficulty uint64
	index      int
}

var _ heap.Interface = &internalTxHeap{}

// internalTxHeap is used to track pending transactions by [difficulty]
type internalTxHeap struct {
	// min-heap pops the lowest difficulty transaction
	// max-heap pops the highest difficulty transaction
	isMinHeap bool

	items  []*txEntry
	lookup map[ids.ID]*txEntry
}

func newInternalTxHeap(items int, isMinHeap bool) *internalTxHeap {
	return &internalTxHeap{
		isMinHeap: isMinHeap,
		items:     make([]*txEntry, 0, items),
		lookup:    map[ids.ID]*txEntry{},
	}
}

func (th internalTxHeap) Len() int { return len(th.items) }

func (th internalTxHeap) Less(i, j int) bool {
	if th.isMinHeap {
		// min-heap pops the lowest difficulty transaction
		return th.items[i].difficulty < th.items[j].difficulty
	}
	// max-heap pops the highest difficulty transaction
	return th.items[i].difficulty > th.items[j].difficulty
}

func (th internalTxHeap) Swap(i, j int) {
	th.items[i], th.items[j] = th.items[j], th.items[i]
	th.items[i].index = i
	th.items[j].index = j
}

func (th *internalTxHeap) Push(x interface{}) {
	entry := x.(*txEntry)
	if th.Has(entry.id) {
		return
	}
	th.items = append(th.items, entry)
	th.lookup[entry.id] = entry
}

func (th *internalTxHeap) Pop() interface{} {
	n := len(th.items)
	item := th.items[n-1]
	th.items[n-1] = nil // avoid memory leak
	th.items = th.items[0 : n-1]
	delete(th.lookup, item.id)
	return item
}

func (th *internalTxHeap) get(id ids.ID) (*txEntry, bool) {
	entry, ok := th.lookup[id]
	if !ok {
		return nil, false
	}
	return entry, true
}

func (th *internalTxHeap) Has(id ids.ID) bool {
	_, has := th.get(id)
	return has
}

var _ memPool = &txHeap{}

// implementing double-ended priority queue
type txHeap struct {
	maxSize int
	maxHeap *internalTxHeap
	minHeap *internalTxHeap
}

func newTxHeap(maxSize int) *txHeap {
	return &txHeap{
		maxSize: maxSize,
		maxHeap: newInternalTxHeap(maxSize, false),
		minHeap: newInternalTxHeap(maxSize, true),
	}
}

func (th *txHeap) push(tx transaction) {
	txID := tx.ID()
	// Don't add duplicates
	if th.has(txID) {
		return
	}
	// Remove the lowest paying tx
	if th.len() >= th.maxSize {
		_ = th.popMin()
	}
	difficulty := tx.Difficulty()
	oldLen := th.len()
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
}

// Assumes there is non-zero items in [txHeap]
func (th *txHeap) peekMax() (transaction, uint64) {
	txEntry := th.maxHeap.items[0]
	return txEntry.tx, txEntry.difficulty
}

// Assumes there is non-zero items in [txHeap]
func (th *txHeap) peekMin() (transaction, uint64) {
	txEntry := th.minHeap.items[0]
	return txEntry.tx, txEntry.difficulty
}

// Assumes there is non-zero items in [txHeap]
func (th *txHeap) popMax() (transaction, uint64) {
	item := th.maxHeap.items[0]
	return th.remove(item.id), item.difficulty
}

// Assumes there is non-zero items in [txHeap]
func (th *txHeap) popMin() transaction {
	return th.remove(th.minHeap.items[0].id)
}

func (th *txHeap) remove(id ids.ID) transaction {
	maxEntry, ok := th.maxHeap.get(id)
	if !ok {
		return nil
	}
	heap.Remove(th.maxHeap, maxEntry.index)

	minEntry, ok := th.minHeap.get(id)
	if !ok {
		// This should never happen, as that would mean the heaps are out of
		// sync.
		return nil
	}
	return heap.Remove(th.minHeap, minEntry.index).(*txEntry).tx
}

func (th *txHeap) prune(validHashes ids.Set) {
	toRemove := []ids.ID{}
	for _, txE := range th.maxHeap.items {
		if !validHashes.Contains(txE.tx.GetBlockID()) {
			toRemove = append(toRemove, txE.id)
		}
	}
	for _, txID := range toRemove {
		th.remove(txID)
	}
}

func (th *txHeap) len() int {
	return th.maxHeap.Len()
}

func (th *txHeap) get(id ids.ID) (transaction, bool) {
	txEntry, ok := th.maxHeap.get(id)
	if !ok {
		return nil, false
	}
	return txEntry.tx, true
}

func (th *txHeap) has(id ids.ID) bool {
	return th.maxHeap.Has(id)
}
