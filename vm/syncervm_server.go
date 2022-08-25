// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package vm implements custom VM.
package vm

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"go.uber.org/zap"
)

// GetLastStateSummary returns the latest state summary.
func (vm *VM) GetLastStateSummary() (block.StateSummary, error) {
	height := vm.lastAccepted.Height()
	summary, err := vm.stateSummaryAtHeight(height)
	vm.ctx.Log.Info(
		"Serving state summary at latest height",
		zap.Uint64("height", height),
		zap.Stringer("summary", summary),
		zap.Error(err))
	return summary, err
}

// GetStateSummary implements StateSyncableVM and returns a summary corresponding
// to the provided [height] if the node can serve state sync data for that key.
// If not, [database.ErrNotFound] must be returned.
func (vm *VM) GetStateSummary(height uint64) (block.StateSummary, error) {
	summary, err := vm.stateSummaryAtHeight(height)
	vm.ctx.Log.Info(
		"Serving state summary at requested height",
		zap.Uint64("height", height),
		zap.Stringer("summary", summary),
		zap.Error(err))
	return summary, err
}

// stateSummaryAtHeight returns the SyncSummary at [height] if valid and available.
func (vm *VM) stateSummaryAtHeight(height uint64) (SyncSummary, error) {
	block, ok := vm.acceptedBlocksByHeight[height]
	if !ok {
		return SyncSummary{}, database.ErrNotFound
	}
	root, ok := vm.acceptedRootsByHeight[height]
	if !ok {
		return SyncSummary{}, database.ErrNotFound
	}
	return NewSyncSummary(block.ID(), height, root)
}
