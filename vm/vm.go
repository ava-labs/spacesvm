// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package vm implements custom VM.
package vm

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ava-labs/avalanchego/cache"
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

	defaultBuildInterval    = 500 * time.Millisecond
	defaultGossipInterval   = 1 * time.Second
	defaultRegossipInterval = 30 * time.Second

	defaultPruneLimit        = 128
	defaultPruneInterval     = time.Minute
	defaultFullPruneInterval = time.Second

	defaultMinimumDifficulty = chain.MinDifficulty
	defaultMinBlockCost      = chain.MinBlockCost

	mempoolSize = 1024
)

var (
	_ snowmanblock.ChainVM = &VM{}
	_ chain.VM             = &VM{}
)

type VM struct {
	ctx *snow.Context
	db  database.Database

	buildInterval    time.Duration
	gossipInterval   time.Duration
	regossipInterval time.Duration

	pruneLimit        int
	pruneInterval     time.Duration
	fullPruneInterval time.Duration

	mempool   *mempool.Mempool
	appSender common.AppSender
	network   *PushNetwork

	// cache block objects to optimize "getBlock"
	// only put when a block is accepted
	// key: block ID, value: *chain.StatelessBlock
	blocks *cache.LRU

	// Block ID --> Block
	// Each element is a block that passed verification but
	// hasn't yet been accepted/rejected
	verifiedBlocks map[ids.ID]*chain.StatelessBlock

	toEngine chan<- common.Message
	builder  BlockBuilder

	preferred    ids.ID
	lastAccepted *chain.StatelessBlock

	minDifficulty uint64
	minBlockCost  uint64

	// beneficiary is the prefix that will receive rewards if the node produces
	// a block
	beneficiaryLock sync.RWMutex
	beneficiary     []byte

	stop chan struct{}

	builderStop chan struct{}
	doneBuild   chan struct{}
	doneGossip  chan struct{}

	donePrune chan struct{}
}

const (
	blocksLRUSize = 100
)

// implements "snowmanblock.ChainVM.common.VM"
func (vm *VM) Initialize(
	ctx *snow.Context,
	dbManager manager.Manager,
	genesisBytes []byte,
	upgradeBytes []byte,
	configBytes []byte,
	toEngine chan<- common.Message,
	_ []*common.Fx,
	appSender common.AppSender,
) error {
	log.Info("initializing quarkvm", "version", version.Version)

	vm.ctx = ctx
	vm.db = dbManager.Current().Database

	// Init channels before initializing other structs
	vm.stop = make(chan struct{})
	vm.builderStop = make(chan struct{})
	vm.doneBuild = make(chan struct{})
	vm.doneGossip = make(chan struct{})
	vm.donePrune = make(chan struct{})

	// TODO: make this configurable via config
	vm.buildInterval = defaultBuildInterval
	vm.gossipInterval = defaultGossipInterval
	vm.regossipInterval = defaultRegossipInterval
	vm.pruneLimit = defaultPruneLimit
	vm.pruneInterval = defaultPruneInterval
	vm.fullPruneInterval = defaultFullPruneInterval

	// TODO: make this configurable via genesis
	vm.minDifficulty, vm.minBlockCost = defaultMinimumDifficulty, defaultMinBlockCost

	vm.mempool = mempool.New(mempoolSize)
	vm.appSender = appSender
	vm.network = vm.NewPushNetwork()

	vm.blocks = &cache.LRU{Size: blocksLRUSize}
	vm.verifiedBlocks = make(map[ids.ID]*chain.StatelessBlock)

	vm.toEngine = toEngine
	vm.builder = vm.NewTimeBuilder()

	// Try to load last accepted
	has, err := chain.HasLastAccepted(vm.db)
	if err != nil {
		log.Error("could not determine if have last accepted")
		return err
	}
	if has { //nolint:nestif
		blkID, err := chain.GetLastAccepted(vm.db)
		if err != nil {
			log.Error("could not get last accepted", "err", err)
			return err
		}

		blk, err := vm.getBlock(blkID)
		if err != nil {
			log.Error("could not load last accepted", "err", err)
			return err
		}

		vm.preferred, vm.lastAccepted = blkID, blk
		log.Info("initialized quarkvm from last accepted", "block", blkID)
	} else {
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
		vm.preferred, vm.lastAccepted = gBlkID, genesisBlk
		log.Info("initialized quarkvm from genesis", "block", gBlkID)
	}

	go vm.builder.Build()
	go vm.builder.Gossip()
	go vm.prune()
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
	close(vm.stop)
	<-vm.doneBuild
	<-vm.doneGossip
	<-vm.donePrune
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

func (vm *VM) getBlock(blkID ids.ID) (*chain.StatelessBlock, error) {
	// has the block been cached from previous "Accepted" call
	bi, exist := vm.blocks.Get(blkID)
	if exist {
		blk, ok := bi.(*chain.StatelessBlock)
		if !ok {
			return nil, fmt.Errorf("unexpected entry %T found in LRU cache, expected *chain.StatelessBlock", bi)
		}
		return blk, nil
	}

	// has the block been verified, not yet accepted
	if blk, exists := vm.verifiedBlocks[blkID]; exists {
		return blk, nil
	}

	// not found in memory, fetch from disk if accepted
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
// called via "avalanchego" node over RPC
func (vm *VM) BuildBlock() (snowman.Block, error) {
	log.Debug("BuildBlock triggered")
	blk, err := chain.BuildBlock(vm, vm.preferred)
	vm.builder.HandleGenerateBlock()
	if err != nil {
		log.Debug("BuildBlock failed", "error", err)
		return nil, err
	}
	sblk, ok := blk.(*chain.StatelessBlock)
	if !ok {
		return nil, fmt.Errorf("unexpected snowman.Block %T, expected *StatelessBlock", blk)
	}

	log.Debug("BuildBlock success",
		"blkID", blk.ID(), "txs", len(sblk.Txs), "beneficiary", string(sblk.Beneficiary),
	)
	return blk, nil
}

func (vm *VM) Submit(txs ...*chain.Transaction) (errs []error) {
	blk, err := vm.GetBlock(vm.preferred)
	if err != nil {
		return []error{err}
	}
	sblk, ok := blk.(*chain.StatelessBlock)
	if !ok {
		return []error{fmt.Errorf("unexpected snowman.Block %T, expected *StatelessBlock", blk)}
	}
	now := time.Now().Unix()
	ctx, err := vm.ExecutionContext(now, sblk)
	if err != nil {
		return []error{err}
	}
	vdb := versiondb.New(vm.db)

	// Expire outdated prefixes before checking submission validity
	if err := chain.ExpireNext(vdb, sblk.Tmstmp, now); err != nil {
		return []error{err}
	}

	for _, tx := range txs {
		if err := vm.submit(tx, vdb, now, ctx); err != nil {
			log.Debug("failed to submit transaction",
				"tx", tx.ID(),
				"error", err,
			)
			errs = append(errs, err)
			continue
		}
		vdb.Abort()
	}
	return errs
}

func (vm *VM) submit(tx *chain.Transaction, db database.Database, blkTime int64, ctx *chain.Context) error {
	if err := tx.Init(); err != nil {
		return err
	}
	if err := tx.ExecuteBase(); err != nil {
		return err
	}
	if err := tx.Execute(db, blkTime, ctx); err != nil {
		return err
	}
	vm.mempool.Add(tx)
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
	return vm.lastAccepted.ID(), nil
}
