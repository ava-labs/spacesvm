// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package timestampvm

import (
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/rpc/v2"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/avalanchego/database/manager"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/utils/json"
	"github.com/ava-labs/avalanchego/version"
)

const (
	dataLen = 32
	Name    = "timestampvm"
)

var (
	errNoPendingBlocks = errors.New("there is no block to propose")
	errBadGenesisBytes = errors.New("genesis data should be bytes (max length 32)")
	Version            = version.NewDefaultVersion(1, 0, 0)

	_ block.ChainVM = &VM{}
)

// VM implements the snowman.VM interface
// Each block in this chain contains a Unix timestamp
// and a piece of data (a string)
type VM struct {
	// The context of this vm
	ctx       *snow.Context
	dbManager manager.Manager

	state State

	// ID of the preferred block
	preferred ids.ID

	// channel to send messages to the consensus engine
	toEngine chan<- common.Message

	// Proposed pieces of data that haven't been put into a block and proposed yet
	mempool       [][dataLen]byte
	currentBlocks map[ids.ID]Block
}

// Initialize this vm
// [ctx] is this vm's context
// [dbManager] is the manager of this vm's database
// [toEngine] is used to notify the consensus engine that new blocks are
//   ready to be added to consensus
// The data in the genesis block is [genesisData]
func (vm *VM) Initialize(
	ctx *snow.Context,
	dbManager manager.Manager,
	genesisData []byte,
	upgradeData []byte,
	configData []byte,
	toEngine chan<- common.Message,
	_ []*common.Fx,
	_ common.AppSender,
) error {
	version, err := vm.Version()
	if err != nil {
		log.Error("error initializing Timestamp VM: %v", err)
		return err
	}
	log.Info("Initializing Timestamp VM", "Version", version)

	vm.dbManager = dbManager
	vm.ctx = ctx
	vm.toEngine = toEngine
	vm.currentBlocks = make(map[ids.ID]Block)

	vm.state = NewState(vm.dbManager.Current().Database, vm)

	if err := vm.initGenesis(genesisData); err != nil {
		return err
	}

	ctx.Log.Info("initializing last accepted block as %s", vm.state.GetLastAccepted())

	// Build off the most recently accepted block
	return vm.SetPreference(vm.state.GetLastAccepted())
}

// SetDBInitialized marks the database as initialized
func (vm *VM) initGenesis(genesisData []byte) error {
	stateInitialized, err := vm.state.IsInitialized()
	if err != nil {
		return err
	}

	if stateInitialized {
		return nil
	}

	if len(genesisData) > dataLen {
		return errBadGenesisBytes
	}

	// genesisData is a byte slice but each block contains an byte array
	// Take the first [dataLen] bytes from genesisData and put them in an array
	var genesisDataArr [dataLen]byte
	copy(genesisDataArr[:], genesisData)

	// Create the genesis block
	// Timestamp of genesis block is 0. It has no parent.
	genesisBlock, err := vm.NewBlock(ids.Empty, 0, genesisDataArr, time.Unix(0, 0))
	if err != nil {
		log.Error("error while creating genesis block: %v", err)
		return err
	}

	if err := vm.state.PutBlock(genesisBlock); err != nil {
		log.Error("error while saving genesis block: %v", err)
		return err
	}

	// Accept the genesis block
	// Sets [vm.lastAccepted] and [vm.preferred]
	if err := genesisBlock.Accept(); err != nil {
		return fmt.Errorf("error accepting genesis block: %w", err)
	}

	if err := vm.state.SetInitialized(); err != nil {
		return fmt.Errorf("error while setting db to initialized: %w", err)
	}

	vm.state.SetLastAccepted(genesisBlock.ID())

	// Flush VM's database to underlying db
	return vm.state.Commit()
}

// CreateHandlers returns a map where:
// Keys: The path extension for this VM's API (empty in this case)
// Values: The handler for the API
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

// CreateStaticHandlers returns a map where:
// Keys: The path extension for this VM's static API
// Values: The handler for that static API
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

// Health implements the common.VM interface
func (vm *VM) HealthCheck() (interface{}, error) { return nil, nil }

// BuildBlock returns a block that this vm wants to add to consensus
func (vm *VM) BuildBlock() (snowman.Block, error) {
	if len(vm.mempool) == 0 { // There is no block to be built
		return nil, errNoPendingBlocks
	}

	// Get the value to put in the new block
	value := vm.mempool[0]
	vm.mempool = vm.mempool[1:]

	// Notify consensus engine that there are more pending data for blocks
	// (if that is the case) when done building this block
	if len(vm.mempool) > 0 {
		defer vm.NotifyBlockReady()
	}

	// Gets Preferred Block
	preferredIntf, err := vm.GetBlock(vm.preferred)
	if err != nil {
		return nil, fmt.Errorf("couldn't get preferred block: %w", err)
	}
	preferredHeight := preferredIntf.Height()

	// Build the block with preferred height
	block, err := vm.NewBlock(vm.preferred, preferredHeight+1, value, time.Now())
	if err != nil {
		return nil, fmt.Errorf("couldn't build block: %w", err)
	}

	// Verifies block
	if err := block.Verify(); err != nil {
		return nil, err
	}
	return block, nil
}

// NotifyBlockReady tells the consensus engine that a new block
// is ready to be created
func (vm *VM) NotifyBlockReady() {
	select {
	case vm.toEngine <- common.PendingTxs:
	default:
		vm.ctx.Log.Debug("dropping message to consensus engine")
	}
}

// GetBlock implements the snowman.ChainVM interface
func (vm *VM) GetBlock(blkID ids.ID) (snowman.Block, error) { return vm.getBlock(blkID) }

func (vm *VM) getBlock(blkID ids.ID) (Block, error) {
	// If block is in memory, return it.
	if blk, exists := vm.currentBlocks[blkID]; exists {
		return blk, nil
	}

	return vm.state.GetBlock(blkID)
}

// LastAccepted returns the block most recently accepted
func (vm *VM) LastAccepted() (ids.ID, error) { return vm.state.GetLastAccepted(), nil }

// proposeBlock appends [data] to [p.mempool].
// Then it notifies the consensus engine
// that a new block is ready to be added to consensus
// (namely, a block with data [data])
func (vm *VM) proposeBlock(data [dataLen]byte) {
	vm.mempool = append(vm.mempool, data)
	vm.NotifyBlockReady()
}

// ParseBlock parses [bytes] to a snowman.Block
// This function is used by the vm's state to unmarshal blocks saved in state
// and by the consensus layer when it receives the byte representation of a block
// from another node
func (vm *VM) ParseBlock(bytes []byte) (snowman.Block, error) {
	// A new empty block
	block := &timeBlock{}

	// Unmarshal the byte repr. of the block into our empty block
	_, err := Codec.Unmarshal(bytes, block)
	if err != nil {
		return nil, err
	}

	// Initialize the block
	// (Block inherits Initialize from its embedded *core.Block)
	block.Initialize(bytes, choices.Processing, vm)

	// Return the block
	return block, nil
}

// NewBlock returns a new Block where:
// - the block's parent is [parentID]
// - the block's data is [data]
// - the block's timestamp is [timestamp]
func (vm *VM) NewBlock(parentID ids.ID, height uint64, data [dataLen]byte, timestamp time.Time) (Block, error) {
	block := newTimeBlock(parentID, height, data, timestamp)

	// Get the byte representation of the block
	blockBytes, err := Codec.Marshal(codecVersion, block)
	if err != nil {
		return nil, err
	}

	// Initialize the block by providing it with its byte representation
	// and a reference to this VM
	block.Initialize(blockBytes, choices.Processing, vm)
	return block, nil
}

// Shutdown this vm
func (vm *VM) Shutdown() error {
	if ok, err := vm.state.IsInitialized(); !ok {
		return err
	}

	// flush DB
	if err := vm.state.Commit(); err != nil {
		return err
	}

	return vm.state.Close() // close versionDB
}

// SetPreference sets the block with ID [ID] as the preferred block
func (vm *VM) SetPreference(id ids.ID) error {
	vm.preferred = id
	return nil
}

// Bootstrapped marks this VM as bootstrapped
func (vm *VM) Bootstrapped() error { return nil }

// Bootstrapping marks this VM as bootstrapping
func (vm *VM) Bootstrapping() error { return nil }

// Returns this VM's version
func (vm *VM) Version() (string, error) {
	return Version.String(), nil
}

func (vm *VM) Connected(id ids.ShortID) error {
	return nil // noop
}

func (vm *VM) Disconnected(id ids.ShortID) error {
	return nil // noop
}

// This VM doesn't (currently) have any app-specific messages
func (vm *VM) AppGossip(nodeID ids.ShortID, msg []byte) error {
	return nil
}

// This VM doesn't (currently) have any app-specific messages
func (vm *VM) AppRequest(nodeID ids.ShortID, requestID uint32, request []byte) error {
	return nil
}

// This VM doesn't (currently) have any app-specific messages
func (vm *VM) AppResponse(nodeID ids.ShortID, requestID uint32, response []byte) error {
	return nil
}

// This VM doesn't (currently) have any app-specific messages
func (vm *VM) AppRequestFailed(nodeID ids.ShortID, requestID uint32) error {
	return nil
}
