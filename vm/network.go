// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"github.com/ava-labs/avalanchego/cache"
	"github.com/ava-labs/avalanchego/ids"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/spacesvm/chain"
)

const (
	gossipedTxsLRUSize = 512
)

type PushNetwork struct {
	vm          *VM
	gossipedTxs *cache.LRU
}

func (vm *VM) NewPushNetwork() *PushNetwork {
	return &PushNetwork{
		vm:          vm,
		gossipedTxs: &cache.LRU{Size: gossipedTxsLRUSize},
	}
}

func (n *PushNetwork) sendTxs(txs []*chain.Transaction) error {
	if len(txs) == 0 {
		return nil
	}

	b, err := chain.Marshal(txs)
	if err != nil {
		log.Warn("failed to marshal txs", "error", err)
		return err
	}

	log.Debug("sending AppGossip",
		"txs", len(txs),
		"size", len(b),
	)
	if err := n.vm.appSender.SendAppGossip(b); err != nil {
		log.Warn(
			"GossipTxs failed",
			"error", err,
		)
		return err
	}

	return nil
}

func (n *PushNetwork) GossipNewTxs(newTxs []*chain.Transaction) error {
	if n.vm.appSender == nil {
		return nil
	}
	txs := []*chain.Transaction{}
	// Gossip at most the target units of a block at once
	for _, tx := range newTxs {
		// skip if recently gossiped
		// to further protect the node from being
		// DDOSed via repeated gossip failures
		if _, exists := n.gossipedTxs.Get(tx.ID()); exists {
			log.Debug("already gossiped, skipping", "txId", tx.ID())
			continue
		}
		n.gossipedTxs.Put(tx.ID(), nil)
		txs = append(txs, tx)
	}

	return n.sendTxs(txs)
}

// Triggers "AppGossip" on the pending transactions in the mempool.
// "force" is true to re-gossip whether recently gossiped or not
func (n *PushNetwork) RegossipTxs() error {
	if n.vm.appSender == nil {
		return nil
	}
	txs := []*chain.Transaction{}
	units := uint64(0)
	// Gossip at most the target units of a block at once
	for n.vm.mempool.Len() > 0 && units < n.vm.genesis.TargetBlockSize {
		tx, _ := n.vm.mempool.PopMax()

		// Note: when regossiping, we force resend eventhough we may have done it
		// recently.
		n.gossipedTxs.Put(tx.ID(), nil)
		txs = append(txs, tx)
		units += tx.LoadUnits(n.vm.genesis)
	}

	return n.sendTxs(txs)
}

// Handles incoming "AppGossip" messages, parses them to transactions,
// and submits them to the mempool. The "AppGossip" message is sent by
// the other VM (spacesvm)  via "common.AppSender" to receive txs and
// forward them to the other node (validator).
//
// implements "snowmanblock.ChainVM.commom.VM.AppHandler"
// assume gossip via proposervm has been activated
// ref. "avalanchego/vms/platformvm/network.AppGossip"
// ref. "coreeth/plugin/evm.GossipHandler.HandleEthTxs"
func (vm *VM) AppGossip(nodeID ids.ShortID, msg []byte) error {
	log.Debug("AppGossip message handler",
		"sender", nodeID,
		"receiver", vm.ctx.NodeID,
		"bytes", len(msg),
	)

	txs := make([]*chain.Transaction, 0)
	if _, err := chain.Unmarshal(msg, &txs); err != nil {
		log.Debug(
			"AppGossip provided invalid txs",
			"peerID", nodeID,
			"err", err,
		)
		return nil
	}

	// submit incoming gossip
	log.Debug("AppGossip transactions are being submitted", "txs", len(txs))
	if errs := vm.Submit(txs...); len(errs) > 0 {
		for _, err := range errs {
			log.Debug(
				"AppGossip failed to submit txs",
				"peerID", nodeID,
				"err", err,
			)
		}
	}

	// only trace error to prevent VM's being shutdown
	// from "AppGossip" returning an error
	// TODO: gracefully handle "AppGossip" failures?
	return nil
}

// used for testing VM
func (vm *VM) Network() *PushNetwork {
	return vm.network
}
