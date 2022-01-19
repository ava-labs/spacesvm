// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"time"

	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	log "github.com/inconshreveable/log15"
)

func BuildBlock(vm VM, preferred ids.ID) (snowman.Block, error) {
	g := vm.Genesis()

	log.Debug("attempting block building")
	nextTime := time.Now().Unix()
	parent, err := vm.GetStatelessBlock(preferred)
	if err != nil {
		log.Debug("block building failed: couldn't get parent", "err", err)
		return nil, err
	}
	context, err := vm.ExecutionContext(nextTime, parent)
	if err != nil {
		log.Debug("block building failed: couldn't get execution context", "err", err)
		return nil, err
	}
	b := NewBlock(vm, parent, nextTime, context)

	// Clean out invalid txs
	mempool := vm.Mempool()
	mempool.Prune(context.RecentBlockIDs)

	parentDB, err := parent.onAccept()
	if err != nil {
		log.Debug("block building failed: couldn't get parent db", "err", err)
		return nil, err
	}
	vdb := versiondb.New(parentDB)

	// Remove all expired spaces
	if err := ExpireNext(vdb, parent.Tmstmp, b.Tmstmp, true); err != nil {
		return nil, err
	}

	b.Winners = map[ids.ID]*Activity{}
	b.Txs = []*Transaction{}
	units := uint64(0)

	// Restorable txs after block attempt finishes
	unusableTxs := []*Transaction{}
	defer func() {
		for _, tx := range unusableTxs {
			mempool.Add(tx)
		}
	}()

	for mempool.Len() > 0 {
		next, price := mempool.PopMax()
		if price < b.Price {
			mempool.Add(next)
			log.Debug("skipping tx: too low price", "block price", b.Price, "tx price", price)
			break
		}
		nextLoad := next.LoadUnits(g)
		if units+nextLoad > g.MaxBlockSize {
			unusableTxs = append(unusableTxs, next)
			log.Debug("skipping tx: too large", "block size", units, "tx load", nextLoad)
			continue // could be txs that fit that are smaller
		}
		// Verify that changes pass
		tvdb := versiondb.New(vdb)
		if err := next.Execute(g, tvdb, b, context); err != nil {
			log.Debug("skipping tx: failed verification", "err", err)
			continue
		}
		if err := tvdb.Commit(); err != nil {
			return nil, err
		}
		// Wait to add spaces until after verification
		b.Txs = append(b.Txs, next)
		units += nextLoad
	}
	vdb.Abort()

	// Compute block hash and marshaled representation
	if err := b.init(); err != nil {
		return nil, err
	}

	// Verify block to ensure it is formed correctly (don't save)
	_, _, err = b.verify()
	if err != nil {
		log.Debug("block building failed: failed verification", "err", err)
		return nil, err
	}
	return b, nil
}
