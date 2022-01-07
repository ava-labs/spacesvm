// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package vm implements custom VM.
package vm

import (
	"fmt"
	"net/http"
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

	defaultWorkInterval     = 100 * time.Millisecond
	defaultRegossipInterval = time.Second
	defaultPruneInterval    = time.Minute

	mempoolSize = 1024
)

var (
	_ snowmanblock.ChainVM = &VM{}
	_ chain.VM             = &VM{}
)

type VM struct {
	ctx *snow.Context
	db  database.Database

	workInterval     time.Duration
	regossipInterval time.Duration
	pruneInterval    time.Duration

	mempool     *mempool.Mempool
	appSender   common.AppSender
	gossipedTxs *cache.LRU
	// cache block objects to optimize "getBlock"
	// only put when a block is accepted
	// key: block ID, value: *chain.StatelessBlock
	blocks *cache.LRU

	// Block ID --> Block
	// Each element is a block that passed verification but
	// hasn't yet been accepted/rejected
	verifiedBlocks map[ids.ID]*chain.StatelessBlock

	toEngine chan<- common.Message
	// signaled when "BuildBlock" is triggered by the engine
	blockBuilder chan struct{}

	preferred    ids.ID
	lastAccepted *chain.StatelessBlock

	// to be set via genesis block
	minDifficulty uint64
	minBlockCost  uint64
	minExpiry     uint64

	stopc         chan struct{}
	donecRun      chan struct{}
	donecRegossip chan struct{}
	donecPrune    chan struct{}
}

const (
	gossipedTxsLRUSize = 512
	blocksLRUSize      = 100
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

	// TODO: make this configurable via config
	vm.workInterval = defaultWorkInterval
	vm.regossipInterval = defaultRegossipInterval

	// to be updated if set in genesis block
	vm.minDifficulty = chain.DefaultMinDifficulty
	vm.minBlockCost = chain.DefaultMinBlockCost
	vm.minExpiry = chain.DefaultMinExpiryTime
	vm.pruneInterval = time.Duration(chain.DefaultPruneInterval) * time.Second

	vm.mempool = mempool.New(mempoolSize)
	vm.appSender = appSender
	vm.gossipedTxs = &cache.LRU{Size: gossipedTxsLRUSize}
	vm.blocks = &cache.LRU{Size: blocksLRUSize}

	vm.verifiedBlocks = make(map[ids.ID]*chain.StatelessBlock)

	vm.toEngine = toEngine
	vm.blockBuilder = make(chan struct{}, 1)

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
		if len(genesisBlk.ExtraData) > 0 {
			genesisExtraData := new(chain.Genesis)
			if _, err := chain.Unmarshal(genesisBlk.ExtraData, genesisExtraData); err != nil {
				log.Error("could not parse genesis extra data", "err", err)
				return err
			}
			if genesisExtraData.MinBlockCost > 0 {
				vm.minBlockCost = genesisExtraData.MinBlockCost
			}
			if genesisExtraData.MinDifficulty > 0 {
				vm.minDifficulty = genesisExtraData.MinDifficulty
			}
			if genesisExtraData.MinExpiry > 0 {
				vm.minExpiry = genesisExtraData.MinExpiry
			}
			if genesisExtraData.PruneInterval > 0 {
				vm.pruneInterval = time.Duration(genesisExtraData.PruneInterval) * time.Second
			}
		}

		gBlkID := genesisBlk.ID()
		vm.preferred, vm.lastAccepted = gBlkID, genesisBlk
		log.Info("initialized quarkvm from genesis", "block", gBlkID)
	}

	vm.stopc = make(chan struct{})
	vm.donecRun = make(chan struct{})
	vm.donecRegossip = make(chan struct{})
	vm.donecPrune = make(chan struct{})

	go vm.run()
	go vm.regossip()
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
	close(vm.stopc)
	<-vm.donecRun
	<-vm.donecRegossip
	<-vm.donecPrune
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
	if err != nil {
		log.Warn("BuildBlock failed", "error", err)
	} else {
		log.Debug("BuildBlock success", "blockId", blk.ID())
	}
	select {
	case vm.blockBuilder <- struct{}{}:
	default:
	}
	return blk, err
}

func (vm *VM) Submit(txs ...*chain.Transaction) (errs []error) {
	blk, err := vm.GetBlock(vm.preferred)
	if err != nil {
		return []error{err}
	}
	now := time.Now().Unix()
	ctx, err := vm.ExecutionContext(now, blk.(*chain.StatelessBlock))
	if err != nil {
		return []error{err}
	}
	vdb := versiondb.New(vm.db)
	defer vdb.Close() // TODO: need to do everywhere?

	for _, tx := range txs {
		if serr := vm.submit(tx, vdb, now, ctx); serr != nil {
			log.Debug("failed to submit transaction",
				"tx", tx.ID(),
				"error", serr,
			)
			errs = append(errs, serr)
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
