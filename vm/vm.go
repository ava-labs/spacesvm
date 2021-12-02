// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package vm implements custom VM.
package vm

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/ava-labs/avalanchego/database/manager"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	snowmanblock "github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/utils/json"
	"github.com/ava-labs/quarkvm/block"
	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/pow"
	"github.com/ava-labs/quarkvm/storage"
	"github.com/ava-labs/quarkvm/transaction"
	"github.com/ava-labs/quarkvm/version"
	"github.com/gorilla/rpc/v2"
	log "github.com/inconshreveable/log15"
)

const Name = "quarkvm"

var _ snowmanblock.ChainVM = &VM{}

var (
	ErrNoPendingTx = errors.New("no pending tx")
)

// TODO: add separate chain state manager?
// TODO: add mutex?

type VM struct {
	ctx          *snow.Context
	sybilControl pow.Checker
	s            storage.Storage
	mempool      transaction.Mempool

	// Block ID --> Block
	// Each element is a block that passed verification but
	// hasn't yet been accepted/rejected
	verifiedBlocks map[ids.ID]*block.Block

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

	// TODO: check initialize from singleton store

	vm.ctx = ctx
	vm.sybilControl = pow.New(vm.getDifficulty)
	vm.s = storage.New(
		ctx,
		dbManager.Current().Database,
	)
	vm.mempool = transaction.NewMempool(1024)
	vm.verifiedBlocks = make(map[ids.ID]*block.Block)
	vm.toEngine = toEngine

	// parent ID, height, timestamp all zero by default
	genesisBlk := new(block.Block)
	genesisBlk.Update(
		genesisBytes,
		choices.Processing,
		vm.s,
		func(id ids.ID) (*block.Block, error) { // lookup
			return vm.getBlock(id)
		},
		func(b *block.Block) error { // persist
			if err := vm.putBlock(b); err != nil {
				return err
			}
			return vm.s.Commit()
		},
		func(id ids.ID) { // set last accepted
			vm.lastAccepted = id
		},
		func(b *block.Block) { // on verified
			vm.verifiedBlocks[b.ID()] = b
		},
	)

	if err := vm.putBlock(genesisBlk); err != nil {
		return err
	}
	if err := genesisBlk.Accept(); err != nil {
		return err
	}
	vm.lastAccepted = genesisBlk.ID()
	vm.preferred = genesisBlk.ID()

	// TODO: set initialize for singleton store

	log.Info("successfully initialized quarkvm")
	return vm.s.Commit()
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
	if err := vm.s.Commit(); err != nil {
		return err
	}
	return vm.s.Close()
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
			LockOptions: common.NoLock,
			Handler:     server,
		},
	}, nil
}

// implements "snowmanblock.ChainVM.common.VM"
// for "ext/vm/[vmID]"
func (vm *VM) CreateStaticHandlers() (map[string]*common.HTTPHandler, error) {
	server := rpc.NewServer()
	server.RegisterCodec(json.NewCodec(), "application/json")
	server.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")
	if err := server.RegisterService(&StaticService{}, Name); err != nil {
		return nil, err
	}
	return map[string]*common.HTTPHandler{
		"": {
			LockOptions: common.NoLock,
			Handler:     server,
		},
	}, nil
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

func (vm *VM) getBlock(blkID ids.ID) (*block.Block, error) {
	if blk, exists := vm.verifiedBlocks[blkID]; exists {
		return blk, nil
	}

	blkBytes, err := vm.s.Block().Get(blkID[:])
	if err != nil {
		return nil, err
	}
	blk := new(block.Block)
	if _, err := codec.Unmarshal(blkBytes, blk); err != nil {
		return nil, err
	}
	return blk, nil
}

// implements "snowmanblock.ChainVM.commom.VM.Parser"
// replaces "core.SnowmanVM.ParseBlock"
func (vm *VM) ParseBlock(source []byte) (snowman.Block, error) {
	blk := new(block.Block)
	if _, err := codec.Unmarshal(source, blk); err != nil {
		return nil, err
	}
	blk.Update(
		source,
		choices.Processing,
		vm.s,
		func(id ids.ID) (*block.Block, error) { // lookup
			return vm.getBlock(id)
		},
		func(b *block.Block) error { // persist
			if err := vm.putBlock(b); err != nil {
				return err
			}
			return vm.s.Commit()
		},
		func(id ids.ID) { // set last accepted
			vm.lastAccepted = id
		},
		func(b *block.Block) { // on verified
			vm.verifiedBlocks[b.ID()] = b
		},
	)
	return blk, nil
}

// TODO: move all this to "chain" own package
// TODO: optimize using "getRecent"?
// TODO: make batch size configurable
// TODO: check min difficulty?
// TODO: check duplicate prefix?
// TODO: what if two different writes with different owners?

// implements "snowmanblock.ChainVM"
func (vm *VM) BuildBlock() (snowman.Block, error) {
	log.Info("building block", "pending-txs", vm.mempool.Len())
	if vm.mempool.Len() == 0 {
		return nil, ErrNoPendingTx
	}

	// for simplicity, just batch as much as we can
	txs := make([]*transaction.Transaction, 0)
	for len(txs) < 100 && vm.mempool.Len() > 0 {
		next, _ := vm.mempool.PopMax()
		txs = append(txs, next)
	}
	if vm.mempool.Len() > 0 {
		// need more block for pending transactions
		defer vm.notifyBlockReady()
	}

	prefBlk, err := vm.getBlock(vm.preferred)
	if err != nil {
		return nil, fmt.Errorf("couldn't get preferred block: %w", err)
	}
	preferredHeight := prefBlk.Height()

	b := &block.Block{
		Prnt:   vm.preferred,
		Tmstmp: time.Now().Unix(),
		Hght:   preferredHeight + 1,
		Txs:    txs,
	}
	b.Update(
		b.Bytes(),
		choices.Processing,
		vm.s,
		func(id ids.ID) (*block.Block, error) { // lookup
			return vm.getBlock(id)
		},
		func(b *block.Block) error { // persist
			if err := vm.putBlock(b); err != nil {
				return err
			}
			return vm.s.Commit()
		},
		func(id ids.ID) { // set last accepted
			vm.lastAccepted = id
		},
		func(b *block.Block) { // on verified
			vm.verifiedBlocks[b.ID()] = b
		},
	)
	log.Info("creating block", "height", b.Hght, "total-txs", len(b.Txs))
	if err := b.Verify(); err != nil {
		return nil, err
	}
	vm.verifiedBlocks[b.ID()] = b

	log.Info("built block")
	return b, nil
}

// "SetPreference" implements "snowmanblock.ChainVM"
// replaces "core.SnowmanVM.SetPreference"
func (vm *VM) SetPreference(id ids.ID) error {
	vm.preferred = id
	return nil
}

// "LastAccepted" implements "snowmanblock.ChainVM"
// replaces "core.SnowmanVM.LastAccepted"
func (vm *VM) LastAccepted() (ids.ID, error) {
	return vm.lastAccepted, nil
}

func (vm *VM) putBlock(b *block.Block) error {
	id := b.ID()
	d, err := codec.Marshal(b)
	if err != nil {
		return err
	}
	return vm.s.Block().Put(id[:], d)
}

func (vm *VM) getDifficulty() uint64 {
	rand.Seed(int64(time.Now().Nanosecond()))
	diff := uint64(rand.Int63n(100))
	return diff
}

func (vm *VM) notifyBlockReady() {
	select {
	case vm.toEngine <- common.PendingTxs:
	default:
		log.Debug("dropping message to consensus engine")
	}
}

func (vm *VM) isTxConfirmed(txID ids.ID) bool {
	id := append([]byte{}, txID[:]...)
	has, err := vm.s.Tx().Has(id)
	if err != nil {
		panic(err)
	}
	return has
}
