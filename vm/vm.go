// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package vm implements custom VM.
package vm

import (
	"errors"
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
	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/storage"
	"github.com/ava-labs/quarkvm/version"
	"github.com/gorilla/rpc/v2"
)

var (
	ErrNoPendingBlock = errors.New("no pending block")
)

var _ snowmanblock.ChainVM = &VM{}

type VM struct {
	ctx      *snow.Context
	s        storage.Storage
	toEngine chan<- common.Message
	chain    chain.Chain

	preferred ids.ID
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
	// TODO: check initialize from singleton store

	vm.ctx = ctx
	vm.s = storage.New(ctx, dbManager.Current().Database)
	vm.toEngine = toEngine
	vm.chain = chain.New(vm.s)

	// parent ID, height, timestamp all zero by default
	genesisBlk := new(block.Block)
	genesisBlk.Update(
		genesisBytes,
		choices.Processing,
		vm.s,
		func(id ids.ID) (*block.Block, error) { // lookup
			return vm.chain.GetBlock(id)
		},
		func(b *block.Block) error { // onVerify
			// TODO: store in the vm.chain block cache
			return nil
		},
		func(b *block.Block) error { // onAccept
			vm.chain.SetLastAccepted(b.ID())
			if err := vm.chain.PutBlock(b); err != nil {
				return err
			}
			return vm.s.Commit()
		},
		func(b *block.Block) error { // onReject
			if err := vm.chain.PutBlock(b); err != nil {
				return err
			}
			return vm.s.Commit()
		},
	)
	if err := vm.chain.PutBlock(genesisBlk); err != nil {
		return err
	}
	if err := genesisBlk.Accept(); err != nil {
		return err
	}
	vm.chain.SetLastAccepted(genesisBlk.ID())

	// TODO: set initialize for singleton store
	return vm.s.Commit()
}

// implements "snowmanblock.ChainVM.common.VM"
func (vm *VM) Bootstrapping() error {
	return nil
}

// implements "snowmanblock.ChainVM.common.VM"
func (vm *VM) Bootstrapped() error {
	vm.ctx.Bootstrapped()
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
func (vm *VM) CreateStaticHandlers() (map[string]*common.HTTPHandler, error) {
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
func (vm *VM) CreateHandlers() (map[string]*common.HTTPHandler, error) {
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
	// (currently) no app-specific messages
	return nil
}

// implements "snowmanblock.ChainVM.commom.VM.health.Checkable"
func (vm *VM) HealthCheck() (interface{}, error) { return "", nil }

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
func (vm *VM) GetBlock(id ids.ID) (snowman.Block, error) {
	// TODO: add cache
	return vm.chain.GetBlock(id)
}

// implements "snowmanblock.ChainVM.commom.VM.Parser"
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
			return vm.chain.GetBlock(id)
		},
		func(b *block.Block) error { // onVerify
			// TODO: store in the vm.chain block cache
			return nil
		},
		func(b *block.Block) error { // onAccept
			vm.chain.SetLastAccepted(b.ID())
			if err := vm.chain.PutBlock(b); err != nil {
				return err
			}
			return vm.s.Commit()
		},
		func(b *block.Block) error { // onReject
			if err := vm.chain.PutBlock(b); err != nil {
				return err
			}
			return vm.s.Commit()
		},
	)
	return blk, nil
}

// implements "snowmanblock.ChainVM"
func (vm *VM) BuildBlock() (snowman.Block, error) {
	// TODO; check pending transactions
	// if vm.chain.Pending() == 0 {
	// 	return nil, ErrNoPendingBlock
	// }

	b := vm.chain.Produce()
	if err := b.Verify(); err != nil {
		return nil, err
	}
	vm.chain.AddBlock(b)
	go func() {
		select {
		case vm.toEngine <- common.PendingTxs:
		default:
			vm.ctx.Log.Debug("dropping message to consensus engine")
		}
	}()
	return b, nil
}

// implements "snowmanblock.ChainVM"
func (vm *VM) SetPreference(id ids.ID) error {
	vm.preferred = id
	return nil
}

// implements "snowmanblock.ChainVM"
func (vm *VM) LastAccepted() (ids.ID, error) {
	return vm.chain.GetLastAccepted(), nil
}

func (s *Service) Put(_ *http.Request, args *PutArgs, reply *PutReply) error {
	if err := s.vm.Put(args); err != nil {
		s.vm.ctx.Log.Warn("failed put for %q (%v)", args.Key, err)
		return err
	}
	reply.Success = true
	return nil
}
