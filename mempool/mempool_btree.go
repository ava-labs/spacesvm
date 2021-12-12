// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mempool

import (
	"bytes"
	"fmt"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/quarkvm/chain"
	"github.com/google/btree"
)

var _ chain.Mempool = &Btree{}

var _ btree.Item = &txDifficulty{}

type txDifficulty struct {
	id         ids.ID
	difficulty uint64
}

func (d *txDifficulty) Less(item btree.Item) bool {
	itemTyped, ok := item.(*txDifficulty)
	if !ok {
		panic(fmt.Errorf("unexpected item found in index %T", item))
	}
	if d.difficulty == itemTyped.difficulty {
		return bytes.Compare(d.id[:], itemTyped.id[:]) == -1
	}
	return d.difficulty < itemTyped.difficulty
}

type Btree struct {
	maxSize int
	txs     map[ids.ID]*chain.Transaction
	tree    *btree.BTree
}

// New creates a new [Mempool]. [maxSize] must be > 0 or else the
// implementation may panic.
func NewBtree(maxSize int) *Btree {
	return &Btree{
		maxSize: maxSize,
		txs:     make(map[ids.ID]*chain.Transaction),
		tree:    btree.New(16),
	}
}

func (mp *Btree) Len() int { return len(mp.txs) }

// Prune removes all transactions that are not found in "validHashes".
func (mp *Btree) Prune(validHashes ids.Set) {
	toRemove := ids.NewSet(len(validHashes))
	for _, tx := range mp.txs {
		txID, blkID := tx.ID(), tx.GetBlockID()
		if !validHashes.Contains(blkID) {
			toRemove.Add(txID)
		}
	}
	if toRemove.Len() == 0 {
		return
	}

	for id := range toRemove {
		item := &txDifficulty{id: id, difficulty: mp.txs[id].Difficulty()}
		mp.tree.Delete(item)
		delete(mp.txs, id)
	}
}

func (mp *Btree) PopMax() (*chain.Transaction, uint64) {
	item := mp.tree.DeleteMax()
	itemTyped, ok := item.(*txDifficulty)
	if !ok {
		panic(fmt.Errorf("unexpected item found in index %T", item))
	}
	tx, ok := mp.txs[itemTyped.id]
	if !ok {
		panic(fmt.Errorf("item not found in mempool %s", itemTyped.id))
	}
	delete(mp.txs, itemTyped.id)
	return tx, itemTyped.difficulty
}

func (mp *Btree) deleteMin() ids.ID {
	item := mp.tree.DeleteMin()
	itemTyped, ok := item.(*txDifficulty)
	if !ok {
		panic(fmt.Errorf("unexpected item found in index %T", item))
	}
	delete(mp.txs, itemTyped.id)
	return itemTyped.id
}

// Returns "true" if the transaction is added.
func (mp *Btree) Add(tx *chain.Transaction) bool {
	txID := tx.ID()
	if _, ok := mp.txs[txID]; ok {
		return false
	}

	diff := tx.Difficulty()
	entry := &txDifficulty{
		id:         txID,
		difficulty: diff,
	}

	mp.txs[txID] = tx
	mp.tree.ReplaceOrInsert(entry)

	// remove lowest paying gas tx
	if len(mp.txs) >= mp.maxSize {
		if minID := mp.deleteMin(); minID == txID {
			return false
		}
	}
	return true
}
