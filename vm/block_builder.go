// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"time"

	"github.com/ava-labs/avalanchego/snow/engine/common"
	log "github.com/inconshreveable/log15"
)

// signal the avalanchego engine
// to build a block from pending transactions
func (vm *VM) NotifyBlockReady() {
	select {
	case vm.toEngine <- common.PendingTxs:
	default:
		log.Debug("dropping message to consensus engine")
	}
}

const blockBuildTimeout = time.Second

// "batchInterval" waits to gossip more txs until some build block
// timeout in order to avoid unnecessary/redundant gossip
// basically, we shouldn't gossip anything included in the block
// to make this more deterministic, we signal block ready and
// wait until "BuildBlock is triggered" from avalanchego
// mempool is shared between "chain.BuildBlock" and "GossipTxs"
// so once tx is included in the block, it won't be included
// in the following "GossipTxs"
// however, we still need to cache recently gossiped txs
// in "GossipTxs" to further protect the node from being
// DDOSed via repeated gossip failures
func (vm *VM) build() {
	log.Debug("starting build loops")
	defer close(vm.donecBuild)

	t := time.NewTimer(vm.buildInterval)
	defer t.Stop()

	buildBlk := true
	for {
		select {
		case <-t.C:
		case <-vm.stopc:
			return
		}
		t.Reset(vm.buildInterval)
		if vm.mempool.Len() == 0 {
			continue
		}

		// TODO: this is async, verify we aren't currently
		// building a block
		if buildBlk {
			// as soon as we receive at least one transaction
			// triggers "BuildBlock" from avalanchego on the local node
			// ref. "plugin/evm.blockBuilder.markBuilding"
			vm.NotifyBlockReady()

			// wait for this node to build a block
			// rather than trigger gossip immediately
			// TODO: "blockBuilder" read may be stale
			// due to lack of request ID for each "common.PendingTxs"
			// just wait some time for best efforts
			select {
			case <-vm.blockBuilder:
				log.Debug("engine just called BuildBlock")
			case <-time.After(blockBuildTimeout):
				// did not build a block, but still gossip
				log.Debug("timed out waiting for BuildBlock from engine")
			case <-vm.stopc:
				return
			}

			// next iteration should be gossip
			// TODO: deciding to build vs gossip shouldn't be based on this
			buildBlk = false
			continue
		}

		// we shouldn't gossip anything included in the block
		// and it's handled via mempool + block build wait above
		_ = vm.GossipTxs(false)
		buildBlk = true
	}
}

// periodically but less aggressively force-regossip the pending
func (vm *VM) regossip() {
	log.Debug("starting regossip loops")
	defer close(vm.donecRegossip)

	// should retry less aggressively
	t := time.NewTimer(vm.regossipInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
		case <-vm.stopc:
			return
		}
		t.Reset(vm.regossipInterval)
		if vm.mempool.Len() == 0 {
			continue
		}

		_ = vm.GossipTxs(true)
	}
}
