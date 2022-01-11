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
	b := NewBlock(vm, parent, nextTime, vm.Beneficiary(), context)

	// Clean out invalid txs
	mempool := vm.Mempool()
	mempool.Prune(context.RecentBlockIDs)

	parentDB, err := parent.onAccept()
	if err != nil {
		log.Debug("block building failed: couldn't get parent db", "err", err)
		return nil, err
	}
	vdb := versiondb.New(parentDB)

	// Remove all expired prefixes
	if err := ExpireNext(vdb, parent.Tmstmp, b.Tmstmp, true); err != nil {
		return nil, err
	}
	// Reward producer (if [b.Beneficiary] is non-nil)
	if err := Reward(vdb, b.Beneficiary); err != nil {
		return nil, err
	}

	b.Txs = []*Transaction{}
	units := uint64(0)
	for units < TargetUnits && mempool.Len() > 0 {
		next, diff := mempool.PopMax()
		if diff < b.Difficulty {
			mempool.Add(next)
			log.Debug("skipping tx: too low difficulty", "block diff", b.Difficulty, "tx diff", next.Difficulty())
			break
		}
		// Verify that changes pass
		tvdb := versiondb.New(vdb)
		if err := next.Execute(tvdb, b.Tmstmp, context); err != nil {
			log.Debug("skipping tx: failed verification", "err", err)
			continue
		}
		if err := tvdb.Commit(); err != nil {
			return nil, err
		}
		// Wait to add prefix until after verification
		b.Txs = append(b.Txs, next)
		units += next.LoadUnits()
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
