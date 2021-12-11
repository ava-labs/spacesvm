// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package vm implements custom VM.
package vm

import (
	"errors"
	"net/http"
	"strings"
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

	defaultBatchInterval = 100 * time.Millisecond
	mempoolSize          = 1024
)

var (
	_ snowmanblock.ChainVM = &VM{}
	_ chain.VM             = &VM{}
)

type VM struct {
	ctx *snow.Context
	db  database.Database

	batchInterval time.Duration
	mempool       *mempool.Mempool
	appSender     common.AppSender
	gossipedTxs   *cache.LRU

	// Block ID --> Block
	// Each element is a block that passed verification but
	// hasn't yet been accepted/rejected
	verifiedBlocks map[ids.ID]*chain.StatelessBlock

	toEngine   chan<- common.Message
	fromEngine chan struct{}

	preferred    ids.ID
	lastAccepted ids.ID

	minDifficulty uint64
	minBlockCost  uint64

	stopc chan struct{}
	donec chan struct{}
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
	appSender common.AppSender,
) error {
	log.Info("initializing quarkvm", "version", version.Version)

	vm.ctx = ctx
	vm.db = dbManager.Current().Database

	vm.batchInterval = defaultBatchInterval
	vm.mempool = mempool.New(mempoolSize)
	vm.appSender = appSender
	vm.gossipedTxs = &cache.LRU{Size: 512}

	vm.verifiedBlocks = make(map[ids.ID]*chain.StatelessBlock)

	vm.toEngine = toEngine
	vm.fromEngine = make(chan struct{}, 1)

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
	vm.minDifficulty, vm.minBlockCost = genesisBlk.Difficulty, genesisBlk.Cost
	log.Info("initialized quarkvm from genesis", "block", gBlkID)

	vm.stopc = make(chan struct{})
	vm.donec = make(chan struct{})

	go vm.run()
	return nil
}

// Updates the build block/gossip interval.
func (vm *VM) SetBatchInterval(d time.Duration) {
	vm.batchInterval = d
}

// signal the avalanchego engine
// to build a block from pending transactions
func (vm *VM) NotifyBlockReady() {
	select {
	case vm.toEngine <- common.PendingTxs:
	default:
		log.Debug("dropping message to consensus engine")
	}
}

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
func (vm *VM) run() {
	defer close(vm.donec)

	t := time.NewTimer(vm.batchInterval)
	defer t.Stop()

	buildBlk := true
	for {
		select {
		case <-t.C:
		case <-vm.stopc:
			return
		}
		t.Reset(vm.batchInterval)
		if vm.mempool.Len() == 0 {
			continue
		}

		if buildBlk {
			// as soon as we receive at least one transaction
			// triggers "BuildBlock" from avalanchego on the local node
			// ref. "plugin/evm.blockBuilder.markBuilding"
			vm.NotifyBlockReady()

			// wait for this node to build a block
			// rather than trigger gossip immediately
			select {
			case <-vm.fromEngine:
				log.Debug("engine just called BuildBlock")
			case <-vm.stopc:
				return
			}

			// next iteration should be gossip
			buildBlk = false
			continue
		}

		// we shouldn't gossip anything included in the block
		// and it's handled via mempool + block build wait above
		_ = vm.GossipTxs(false)
		buildBlk = true
	}
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
	<-vm.donec
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
	case vm.fromEngine <- struct{}{}:
	default:
	}
	return blk, err
}

func (vm *VM) Submit(txs ...*chain.Transaction) (err error) {
	blk, err := vm.GetBlock(vm.preferred)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	ctx, err := vm.ExecutionContext(now, blk.(*chain.StatelessBlock))
	if err != nil {
		return err
	}
	vdb := versiondb.New(vm.db)
	defer vdb.Close() // TODO: need to do everywhere?

	es := make([]string, 0)
	for _, tx := range txs {
		if serr := vm.submit(tx, vdb, now, ctx); serr != nil {
			log.Warn("failed to submit transaction",
				"tx", tx.ID(),
				"error", serr,
			)
			es = append(es, serr.Error())
			continue
		}
		vdb.Abort()
	}
	if len(es) > 0 {
		return errors.New(strings.Join(es, ","))
	}
	return nil
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
	return vm.lastAccepted, nil
}
