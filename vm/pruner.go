// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"time"

	"github.com/ava-labs/avalanchego/database/versiondb"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/quarkvm/chain"
)

const (
	pruneLimit = 128
)

func (vm *VM) pruneCall() {
	// Lock to prevent concurrent modification of state
	vm.ctx.Lock.Lock()
	defer vm.ctx.Lock.Unlock()

	vdb := versiondb.New(vm.db)
	defer vdb.Abort()
	if err := chain.PruneNext(vdb, pruneLimit); err != nil {
		log.Warn("unable to prune next range", "error", err)
		return
	}
	if err := vdb.Commit(); err != nil {
		log.Warn("unable to commit pruning work", "error", err)
		return
	}
	if err := vm.lastAccepted.SetChildrenDB(vm.db); err != nil {
		log.Error("unable to update child databases of last accepted block", "error", err)
	}
}

func (vm *VM) prune() {
	log.Debug("starting prune loops")
	defer close(vm.donecPrune)

	// should retry less aggressively
	t := time.NewTimer(vm.pruneInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
		case <-vm.stopc:
			return
		}
		t.Reset(vm.pruneInterval)
		vm.pruneCall()
	}
}
