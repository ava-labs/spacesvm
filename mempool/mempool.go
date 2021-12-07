package mempool

import (
	"container/heap"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/quarkvm/chain"
)

// txEntry is used to track the work transactions pay to be included in
// the mempool.
type txEntry struct {
	id         ids.ID
	tx         *chain.Transaction
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
	return entry, ok
}

func (th *internalTxHeap) Has(id ids.ID) bool {
	_, has := th.Get(id)
	return has
}

type Mempool struct {
	maxSize int
	maxHeap *internalTxHeap
	minHeap *internalTxHeap
}

func New(maxSize int) *Mempool {
	return &Mempool{
		maxSize: maxSize,
		maxHeap: newInternalTxHeap(maxSize, false),
		minHeap: newInternalTxHeap(maxSize, true),
	}
}

func (th *Mempool) Push(tx *chain.Transaction) {
	txID := tx.ID()
	// Don't add duplicates
	if th.Has(txID) {
		return
	}
	// Optimistically add tx to mempool
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
	// Remove the lowest paying tx
	//
	// Note: we do this after adding the new transaction in case it is the new
	// lowest paying transaction
	if th.Len() >= th.maxSize {
		_ = th.PopMin()
	}
}

// Assumes there is non-zero items in [Mempool]
func (th *Mempool) PeekMax() (*chain.Transaction, uint64) {
	txEntry := th.maxHeap.items[0]
	return txEntry.tx, txEntry.difficulty
}

// Assumes there is non-zero items in [Mempool]
func (th *Mempool) PeekMin() (*chain.Transaction, uint64) {
	txEntry := th.minHeap.items[0]
	return txEntry.tx, txEntry.difficulty
}

// Assumes there is non-zero items in [Mempool]
func (th *Mempool) PopMax() (*chain.Transaction, uint64) {
	item := th.maxHeap.items[0]
	return th.Remove(item.id), item.difficulty
}

// Assumes there is non-zero items in [Mempool]
func (th *Mempool) PopMin() *chain.Transaction {
	return th.Remove(th.minHeap.items[0].id)
}

func (th *Mempool) Remove(id ids.ID) *chain.Transaction {
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

// TODO: remember to prune
func (th *Mempool) Prune(validHashes ids.Set) {
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

func (th *Mempool) Len() int {
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
	return th.maxHeap.Has(id)
}
