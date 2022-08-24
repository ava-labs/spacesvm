// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package vm implements custom VM.
package vm

import (
	"context"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/merkledb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/sync"
	log "github.com/inconshreveable/log15"
)

const (
	numSyncThreads    = 4
	maxActiveRequests = 10
)

type stateSyncClientConfig struct {
	enabled          bool
	db               database.Database
	stateSyncNodeIDs []ids.NodeID
	networkClient    sync.NetworkClient
	toEngine         chan<- common.Message
}

type stateSyncClient struct {
	*stateSyncClientConfig

	cancel context.CancelFunc

	// State Sync results
	syncSummary  SyncSummary
	stateSyncErr error
}

func NewStateSyncClient(config *stateSyncClientConfig) *stateSyncClient {
	return &stateSyncClient{
		stateSyncClientConfig: config,
	}
}

// StateSyncEnabled returns [client.enabled]
func (client *stateSyncClient) StateSyncEnabled() (bool, error) { return client.enabled, nil }

// GetOngoingSyncStateSummary returns [database.ErrNotFound] since
// we don't support resume in this demo.
func (client *stateSyncClient) GetOngoingSyncStateSummary() (block.StateSummary, error) {
	return nil, database.ErrNotFound
}

// ParseStateSummary parses [summaryBytes] to [commonEng.Summary]
func (client *stateSyncClient) ParseStateSummary(summaryBytes []byte) (block.StateSummary, error) {
	return NewSyncSummaryFromBytes(summaryBytes, client.acceptSyncSummary)
}

// acceptSyncSummary returns true if sync will be performed and launches the state sync process
// in a goroutine.
func (client *stateSyncClient) acceptSyncSummary(summary SyncSummary) (bool, error) {
	log.Info("Starting state sync", "summary", summary)
	client.syncSummary = summary

	go func() {
		if err := client.stateSync(); err != nil {
			client.stateSyncErr = err
		} else {
			client.stateSyncErr = client.finishSync()
		}
		// notify engine regardless of whether err == nil,
		// this error will be propagated to the engine when it calls
		// vm.SetState(snow.Bootstrapping)
		log.Info("stateSync completed, notifying engine", "err", client.stateSyncErr)
		client.toEngine <- common.StateSyncDone
	}()
	return true, nil
}

// stateSync blockingly performs the state sync for [client.syncSummary].
// Returns an error if one occurred.
func (client *stateSyncClient) stateSync() error {
	syncClient := sync.NewClient(&sync.ClientConfig{
		NetworkClient:    client.networkClient,
		StateSyncNodeIDs: client.stateSyncNodeIDs,
		Log:              newLogger("sync-client"),
	})

	worker := sync.NewStateSyncWorker(&sync.StateSyncConfig{
		SyncDB:                client.db.(*merkledb.MerkleDB),
		Client:                syncClient,
		RootID:                client.syncSummary.BlockRoot,
		SimultaneousWorkLimit: numSyncThreads,
		Log:                   newLogger("sync-worker"),
	})

	ctx, cancel := context.WithCancel(context.Background())
	client.cancel = cancel
	if err := worker.StartSyncing(ctx); err != nil {
		return err
	}
	return worker.Wait()
}

// finishSync is called after a successful state sync to update necessary pointers
// for the VM to begin normal operations.
func (client *stateSyncClient) finishSync() error { return nil }

func (client *stateSyncClient) Shutdown() {
	if client.cancel != nil {
		client.cancel()
	}
}
