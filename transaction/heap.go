// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package transaction

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

var _ heap.Interface = &txHeap{}

// txHeap is used to track pending transactions by [difficulty]
type txHeap struct {
	// min-heap pops the lowest difficulty transaction
	// max-heap pops the highest difficulty transaction
	isMinHeap bool

	items  []*txEntry
	lookup map[ids.ID]*txEntry
}

func newTxHeap(items int, isMinHeap bool) *txHeap {
	return &txHeap{
		isMinHeap: isMinHeap,
		items:     make([]*txEntry, 0, items),
		lookup:    map[ids.ID]*txEntry{},
	}
}

func (th txHeap) Len() int { return len(th.items) }

func (th txHeap) Less(i, j int) bool {
	if th.isMinHeap {
		// min-heap pops the lowest difficulty transaction
		return th.items[i].difficulty < th.items[j].difficulty
	}
	// max-heap pops the highest difficulty transaction
	return th.items[i].difficulty > th.items[j].difficulty
}

func (th txHeap) Swap(i, j int) {
	th.items[i], th.items[j] = th.items[j], th.items[i]
	th.items[i].index = i
	th.items[j].index = j
}

func (th *txHeap) Push(x interface{}) {
	entry := x.(*txEntry)
	if th.has(entry.id) {
		return
	}
	th.items = append(th.items, entry)
	th.lookup[entry.id] = entry
}

func (th *txHeap) Pop() interface{} {
	n := len(th.items)
	item := th.items[n-1]
	th.items[n-1] = nil // avoid memory leak
	th.items = th.items[0 : n-1]
	delete(th.lookup, item.id)
	return item
}

func (th *txHeap) get(id ids.ID) (*txEntry, bool) {
	entry, ok := th.lookup[id]
	if !ok {
		return nil, false
	}
	return entry, true
}

func (th *txHeap) has(id ids.ID) bool {
	_, has := th.get(id)
	return has
}
