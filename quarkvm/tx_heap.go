package quarkvm

import (
	"container/heap"

	"github.com/ava-labs/avalanchego/ids"
)

// txEntry is used to track the work transactions pay to be included in
// the mempool.
type txEntry struct {
	id         ids.ID
	tx         *Transaction
	difficulty uint64
	index      int
}

// internalTxHeap is used to track pending transactions by [difficulty]
type internalTxHeap struct {
	isMinHeap bool
	items     []*txEntry
	lookup    map[ids.ID]*txEntry
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
		return th.items[i].difficulty < th.items[j].difficulty
	}
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

func (th *internalTxHeap) Get(id ids.ID) (*txEntry, bool) {
	entry, ok := th.lookup[id]
	if !ok {
		return nil, false
	}
	return entry, true
}

func (th *internalTxHeap) Has(id ids.ID) bool {
	_, has := th.Get(id)
	return has
}

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

func (th *txHeap) Push(tx *Transaction) {
	txID := tx.ID()
	// Don't add duplicates
	if th.Has(txID) {
		return
	}
	// Remove the lowest paying tx
	if th.Len() >= th.maxSize {
		_ = th.PopMin()
	}
	difficulty := tx.Difficulty()
	oldLen := th.Len()
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
func (th *txHeap) PeekMax() (*Transaction, uint64) {
	txEntry := th.maxHeap.items[0]
	return txEntry.tx, txEntry.difficulty
}

// Assumes there is non-zero items in [txHeap]
func (th *txHeap) PeekMin() (*Transaction, uint64) {
	txEntry := th.minHeap.items[0]
	return txEntry.tx, txEntry.difficulty
}

// Assumes there is non-zero items in [txHeap]
func (th *txHeap) PopMax() (*Transaction, uint64) {
	item := th.maxHeap.items[0]
	return th.Remove(item.id), item.difficulty
}

// Assumes there is non-zero items in [txHeap]
func (th *txHeap) PopMin() *Transaction {
	return th.Remove(th.minHeap.items[0].id)
}

func (th *txHeap) Remove(id ids.ID) *Transaction {
	maxEntry, ok := th.maxHeap.Get(id)
	if !ok {
		return nil
	}
	heap.Remove(th.maxHeap, maxEntry.index)

	minEntry, ok := th.minHeap.Get(id)
	if !ok {
		// This should never happen, as that would mean the heaps are out of
		// sync.
		return nil
	}
	return heap.Remove(th.minHeap, minEntry.index).(*txEntry).tx
}

func (th *txHeap) Prune(validHashes ids.Set) {
	toRemove := []ids.ID{}
	for _, txE := range th.maxHeap.items {
		if !validHashes.Contains(txE.tx.GetBlockID()) {
			toRemove = append(toRemove, txE.id)
		}
	}
	for _, txID := range toRemove {
		th.Remove(txID)
	}
}

func (th *txHeap) Len() int {
	return th.maxHeap.Len()
}

func (th *txHeap) Get(id ids.ID) (*Transaction, bool) {
	txEntry, ok := th.maxHeap.Get(id)
	if !ok {
		return nil, false
	}
	return txEntry.tx, true
}

func (th *txHeap) Has(id ids.ID) bool {
	return th.maxHeap.Has(id)
}
