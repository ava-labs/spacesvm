// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mempool

import (
	"fmt"

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
	entry, ok := x.(*txEntry)
	if !ok {
		panic(fmt.Errorf("unexpected %T, expected *txEntry", x))
	}
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
