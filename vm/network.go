// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/quarkvm/chain"
	log "github.com/inconshreveable/log15"
)

// Triggers "AppGossip" on the pending transactions in the mempool.
// "force" is true to re-gossip whether recently gossiped or not
func (vm *VM) GossipTxs(force bool) error {
	if vm.appSender == nil {
		return nil
	}
	txs := []*chain.Transaction{}
	for vm.mempool.Len() > 0 && len(txs) < chain.TargetTransactions {
		tx, _ := vm.mempool.PopMax()
		if !force {
			// skip if recently gossiped
			// to further protect the node from being
			// DDOSed via repeated gossip failures
			if _, exists := vm.gossipedTxs.Get(tx.ID()); exists {
				log.Debug("already gossiped, skipping", "txId", tx.ID())
				vm.mempool.Add(tx)
				continue
			}
		}
		vm.gossipedTxs.Put(tx.ID(), nil)
		txs = append(txs, tx)
	}

	b, err := chain.Marshal(txs)
	if err != nil {
		log.Warn("failed to marshal txs", "error", err)
	} else {
		log.Debug("sending AppGossip",
			"txs", len(txs),
			"size", len(b),
		)
		err = vm.appSender.SendAppGossip(b)
	}
	if err == nil {
		return nil
	}

	log.Warn(
		"GossipTxs failed; txs back to mempool",
		"error", err,
	)
	for _, tx := range txs {
		vm.mempool.Add(tx)
	}
	return err
}

// Handles incoming "AppGossip" messages, parses them to transactions,
// and submits them to the mempool. The "AppGossip" message is sent by
// the other VM (quarkvm)  via "common.AppSender" to receive txs and
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
		log.Warn(
			"AppGossip provided invalid txs",
			"peerID", nodeID,
			"err", err,
		)
		return nil
	}

	// submit incoming gossip
	log.Debug("AppGossip transactions are being submitted", "txs", len(txs))
	if err := vm.Submit(txs...); err != nil {
		log.Warn(
			"AppGossip failed to submit txs",
			"peerID", nodeID,
			"err", err,
		)
	}

	// only trace error to prevent VM's being shutdown
	// from "AppGossip" returning an error
	// TODO: gracefully handle "AppGossip" failures?
	return nil
}
