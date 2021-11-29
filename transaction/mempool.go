// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package transaction

import (
	"container/heap"

	"github.com/ava-labs/avalanchego/ids"
)

// Mempool defines in-memory transaction pool.
type Mempool interface {
	Push(tx *Transaction)
	PeekMax() (*Transaction, uint64)
	PeekMin() (*Transaction, uint64)
	PopMax() (*Transaction, uint64)
	PopMin() *Transaction
	Remove(id ids.ID) *Transaction
	Len() int
	Get(id ids.ID) (*Transaction, bool)
	Has(id ids.ID) bool
}

var _ Mempool = &txMempool{}

// implementing double-ended priority queue
type txMempool struct {
	maxSize int
	maxHeap *txHeap
	minHeap *txHeap
}

func NewMempool(maxSize int) Mempool {
	return &txMempool{
		maxSize: maxSize,
		maxHeap: newTxHeap(maxSize, false),
		minHeap: newTxHeap(maxSize, true),
	}
}

func (txm *txMempool) Push(tx *Transaction) {
	txID := tx.ID()
	// Don't add duplicates
	if txm.Has(txID) {
		return
	}
	// Remove the lowest paying tx
	if txm.Len() >= txm.maxSize {
		_ = txm.PopMin()
	}
	difficulty := tx.Difficulty()
	oldLen := txm.Len()
	heap.Push(txm.maxHeap, &txEntry{
		id:         txID,
		difficulty: difficulty,
		tx:         tx,
		index:      oldLen,
	})
	heap.Push(txm.minHeap, &txEntry{
		id:         txID,
		difficulty: difficulty,
		tx:         tx,
		index:      oldLen,
	})
}

// Assumes there is non-zero items in [txHeap]
func (txm *txMempool) PeekMax() (*Transaction, uint64) {
	txEntry := txm.maxHeap.items[0]
	return txEntry.tx, txEntry.difficulty
}

// Assumes there is non-zero items in [txHeap]
func (txm *txMempool) PeekMin() (*Transaction, uint64) {
	txEntry := txm.minHeap.items[0]
	return txEntry.tx, txEntry.difficulty
}

// Assumes there is non-zero items in [txHeap]
func (txm *txMempool) PopMax() (*Transaction, uint64) {
	item := txm.maxHeap.items[0]
	return txm.Remove(item.id), item.difficulty
}

// Assumes there is non-zero items in [txHeap]
func (txm *txMempool) PopMin() *Transaction {
	return txm.Remove(txm.minHeap.items[0].id)
}

func (txm *txMempool) Remove(id ids.ID) *Transaction {
	maxEntry, ok := txm.maxHeap.get(id)
	if !ok {
		return nil
	}
	heap.Remove(txm.maxHeap, maxEntry.index)

	minEntry, ok := txm.minHeap.get(id)
	if !ok {
		// This should never happen, as that would mean the heaps are out of
		// sync.
		return nil
	}
	return heap.Remove(txm.minHeap, minEntry.index).(*txEntry).tx
}

func (txm *txMempool) Len() int {
	return txm.maxHeap.Len()
}

func (txm *txMempool) Get(id ids.ID) (*Transaction, bool) {
	txEntry, ok := txm.maxHeap.get(id)
	if !ok {
		return nil, false
	}
	return txEntry.tx, true
}

func (txm *txMempool) Has(id ids.ID) bool {
	return txm.maxHeap.has(id)
}
