// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"time"

	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/spacesvm/chain"
)

func (vm *VM) compact() {
	log.Debug("starting compaction loops")
	defer close(vm.doneCompact)

	t := time.NewTimer(vm.config.CompactInterval)
	defer t.Stop()

	ranges := chain.CompactableRanges
	currentRange := 0

	// Ensure there is something to compact
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
		start := time.Now()
		prefix := ranges[currentRange]
		if err := vm.db.Compact(prefix.Start, prefix.End); err != nil {
			log.Error("unable to compact prefix range", "start", prefix.Start, "stop", prefix.End)
		}
		log.Debug("compacted prefix", "start", prefix.Start, "stop", prefix.End, "t", time.Since(start))

		// Update range compaction index
		currentRange++
		if currentRange > len(ranges)-1 {
			currentRange = 0
		}

		t.Reset(vm.config.CompactInterval)
	}
}
