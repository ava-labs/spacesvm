// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package vm implements custom VM.
package vm

import (
	"net/http"
	"time"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/manager"
	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	snowmanblock "github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/utils/json"
	"github.com/gorilla/rpc/v2"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/mempool"
	"github.com/ava-labs/quarkvm/version"
)

const (
	Name = "quarkvm"

	mempoolSize = 1024
)

var (
	_ snowmanblock.ChainVM = &VM{}
	_ chain.VM             = &VM{}
)

type VM struct {
	ctx     *snow.Context
	db      database.Database
	mempool *mempool.Mempool

	// Block ID --> Block
	// Each element is a block that passed verification but
	// hasn't yet been accepted/rejected
	verifiedBlocks map[ids.ID]*chain.Block

	toEngine chan<- common.Message

	preferred    ids.ID
	lastAccepted ids.ID
}

// implements "snowmanblock.ChainVM.common.VM"
func (vm *VM) Initialize(
	ctx *snow.Context,
	dbManager manager.Manager,
	genesisBytes []byte,
	upgradeBytes []byte,
	configBytes []byte,
	toEngine chan<- common.Message,
	_ []*common.Fx,
	_ common.AppSender,
) error {
	log.Info("initializing quarkvm", "version", version.Version)

	vm.ctx = ctx
	vm.db = dbManager.Current().Database
	vm.mempool = mempool.New(mempoolSize)
	vm.verifiedBlocks = make(map[ids.ID]*chain.Block)
	vm.toEngine = toEngine

	// Try to load last accepted
	has, err := chain.HasLastAccepted(vm.db)
	if err != nil {
		log.Error("could not determine if have last accepted")
		return err
	}
	if has {
		b, err := chain.GetLastAccepted(vm.db)
		if err != nil {
			log.Error("could not get last accepted", "err", err)
			return err
		}

		vm.preferred = b
		vm.lastAccepted = b
		log.Info("initialized quarkvm from last accepted", "block", b)
		return nil
	}

	// Load from genesis
	genesisBlk, err := chain.ParseBlock(
		genesisBytes,
		choices.Accepted,
		vm,
	)
	if err != nil {
		log.Error("unable to init genesis block", "err", err)
		return err
	}
	if err := chain.SetLastAccepted(vm.db, genesisBlk); err != nil {
		log.Error("could not set genesis as last accepted", "err", err)
		return err
	}
	gBlkID := genesisBlk.ID()
	vm.preferred, vm.lastAccepted = gBlkID, gBlkID
	log.Info("initialized quarkvm from genesis", "block", gBlkID)
	return nil
}

// implements "snowmanblock.ChainVM.common.VM"
func (vm *VM) Bootstrapping() error {
	return nil
}

// implements "snowmanblock.ChainVM.common.VM"
func (vm *VM) Bootstrapped() error {
	return nil
}

// implements "snowmanblock.ChainVM.common.VM"
func (vm *VM) Shutdown() error {
	if vm.ctx == nil {
		return nil
	}
	return vm.db.Close()
}

// implements "snowmanblock.ChainVM.common.VM"
func (vm *VM) Version() (string, error) { return version.Version.String(), nil }

// implements "snowmanblock.ChainVM.common.VM"
// for "ext/vm/[chainID]"
func (vm *VM) CreateHandlers() (map[string]*common.HTTPHandler, error) {
	server := rpc.NewServer()
	server.RegisterCodec(json.NewCodec(), "application/json")
	server.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")
	if err := server.RegisterService(&Service{vm: vm}, Name); err != nil {
		return nil, err
	}
	return map[string]*common.HTTPHandler{
		"": {
			Handler: server,
		},
	}, nil
}

// implements "snowmanblock.ChainVM.common.VM"
// for "ext/vm/[vmID]"
func (vm *VM) CreateStaticHandlers() (map[string]*common.HTTPHandler, error) {
	return nil, nil
}

// implements "snowmanblock.ChainVM.commom.VM.AppHandler"
func (vm *VM) AppRequest(nodeID ids.ShortID, requestID uint32, deadline time.Time, request []byte) error {
	// (currently) no app-specific messages
	return nil
}

// implements "snowmanblock.ChainVM.commom.VM.AppHandler"
func (vm *VM) AppRequestFailed(nodeID ids.ShortID, requestID uint32) error {
	// (currently) no app-specific messages
	return nil
}

// implements "snowmanblock.ChainVM.commom.VM.AppHandler"
func (vm *VM) AppResponse(nodeID ids.ShortID, requestID uint32, response []byte) error {
	// (currently) no app-specific messages
	return nil
}

// implements "snowmanblock.ChainVM.commom.VM.AppHandler"
func (vm *VM) AppGossip(nodeID ids.ShortID, msg []byte) error {
	// TODO: gossip txs
	return nil
}

// implements "snowmanblock.ChainVM.commom.VM.health.Checkable"
func (vm *VM) HealthCheck() (interface{}, error) {
	return http.StatusOK, nil
}

// implements "snowmanblock.ChainVM.commom.VM.validators.Connector"
func (vm *VM) Connected(id ids.ShortID) error {
	// no-op
	return nil
}

// implements "snowmanblock.ChainVM.commom.VM.validators.Connector"
func (vm *VM) Disconnected(id ids.ShortID) error {
	// no-op
	return nil
}

// implements "snowmanblock.ChainVM.commom.VM.Getter"
// replaces "core.SnowmanVM.GetBlock"
func (vm *VM) GetBlock(id ids.ID) (snowman.Block, error) {
	b, err := vm.getBlock(id)
	if err != nil {
		log.Warn("failed to get block", "err", err)
	}
	return b, err
}

func (vm *VM) getBlock(blkID ids.ID) (*chain.Block, error) {
	if blk, exists := vm.verifiedBlocks[blkID]; exists {
		return blk, nil
	}
	bytes, err := chain.GetBlock(vm.db, blkID)
	if err != nil {
		return nil, err
	}
	// If block on disk, it must've been accepted
	return chain.ParseBlock(bytes, choices.Accepted, vm)
}

// implements "snowmanblock.ChainVM.commom.VM.Parser"
// replaces "core.SnowmanVM.ParseBlock"
func (vm *VM) ParseBlock(source []byte) (snowman.Block, error) {
	blk, err := chain.ParseBlock(
		source,
		choices.Processing,
		vm,
	)
	if err != nil {
		log.Error("could not parse block", "err", err)
	} else {
		log.Debug("parsing block", "id", blk.ID())
	}
	return blk, err
}

// implements "snowmanblock.ChainVM"
func (vm *VM) BuildBlock() (snowman.Block, error) {
	return chain.BuildBlock(vm, vm.preferred)
}

func (vm *VM) Submit(tx *chain.Transaction) error {
	if err := tx.Init(); err != nil {
		return err
	}
	blk, err := vm.GetBlock(vm.preferred)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	context, err := vm.ExecutionContext(now, blk.(*chain.Block))
	if err != nil {
		return err
	}
	vdb := versiondb.New(vm.db)
	defer vdb.Close() // TODO: need to do everywhere?
	if err := tx.Execute(vdb, now, context); err != nil {
		return err
	}
	if added := vm.mempool.Add(tx); !added {
		// Don't gossip if not added
		return nil
	}

	// TODO: do on a block timer
	// TODO: wait to gossip if can create a block
	vm.notifyBlockReady()
	return nil
}

// "SetPreference" implements "snowmanblock.ChainVM"
// replaces "core.SnowmanVM.SetPreference"
func (vm *VM) SetPreference(id ids.ID) error {
	log.Debug("set preference", "id", id)
	vm.preferred = id
	return nil
}

// "LastAccepted" implements "snowmanblock.ChainVM"
// replaces "core.SnowmanVM.LastAccepted"
func (vm *VM) LastAccepted() (ids.ID, error) {
	return vm.lastAccepted, nil
}

func (vm *VM) notifyBlockReady() {
	select {
	case vm.toEngine <- common.PendingTxs:
	default:
		log.Debug("dropping message to consensus engine")
	}
}
