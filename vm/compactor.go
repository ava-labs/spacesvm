// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"time"

	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/spacesvm/chain"
)

func (vm *VM) compactCall(prefix *chain.PrefixRange) {
	// Lock to prevent concurrent modification of state
	vm.ctx.Lock.Lock()
	defer vm.ctx.Lock.Unlock()

	start := time.Now()
	if err := vm.db.Compact(prefix.Start, prefix.End); err != nil {
		log.Error("unable to compact prefix range", "start", prefix.Start, "stop", prefix.End)
		return
	}
	log.Debug("compacted prefix", "start", prefix.Start, "stop", prefix.End, "t", time.Since(start))

	// Make sure to update children or else won't be persisted
	if err := vm.lastAccepted.SetChildrenDB(vm.db); err != nil {
		log.Error("unable to update child databases of last accepted block", "error", err)
		return
	}
}

func (vm *VM) compact() {
	log.Debug("starting compaction loops")
	defer close(vm.doneCompact)

	t := time.NewTimer(vm.config.CompactInterval)
	defer t.Stop()

	// Ensure there is something to compact
	ranges := chain.CompactableRanges
	currentRange := 0
	if len(ranges) == 0 {
		log.Debug("exiting compactor because nothing to compact")
		return
	}

	for {
		select {
		case <-t.C:
		case <-vm.stop:
			return
		}

		// Compact next range
		prefix := ranges[currentRange]
		vm.compactCall(prefix)

		// Update range compaction index
		currentRange++
		if currentRange > len(ranges)-1 {
			currentRange = 0
		}

		t.Reset(vm.config.CompactInterval)
	}
}
