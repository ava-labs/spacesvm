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

	prefixes := chain.CompactablePrefixes
	currentPrefix := 0

	for {
		select {
		case <-t.C:
		case <-vm.stop:
			return
		}

		// Compact next range
		start := time.Now()
		rangeStart := chain.CompactablePrefixKey(prefixes[currentPrefix])
		rangeEnd := chain.CompactablePrefixKey(prefixes[currentPrefix] + 1)
		if err := vm.db.Compact(rangeStart, rangeEnd); err != nil {
			log.Error("unable to compact prefix range", "start", string(rangeStart), "stop", string(rangeEnd))
		}
		log.Debug("compacted prefix", "prefix", string(rangeStart), "t", time.Since(start))

		// Update prefix compaction index
		currentPrefix++
		if currentPrefix > len(prefixes)-1 {
			currentPrefix = 0
		}

		t.Reset(vm.config.CompactInterval)
	}
}
