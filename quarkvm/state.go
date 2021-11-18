// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package quarkvm

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/prefixdb"
	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/vms/components/avax"
)

var (
	singletonStatePrefix = []byte("singleton")
	blockStatePrefix     = []byte("block")

	_ State = &state{}
)

type State interface {
	avax.SingletonState
	BlockState

	Commit() error
	Close() error
}

type state struct {
	avax.SingletonState
	BlockState

	baseDB *versiondb.Database
}

func NewState(db database.Database, vm *VM) State {
	baseDB := versiondb.New(db)

	blockDB := prefixdb.New(blockStatePrefix, baseDB)
	singletonDB := prefixdb.New(singletonStatePrefix, baseDB)

	return &state{
		BlockState:     NewBlockState(blockDB, vm),
		SingletonState: avax.NewSingletonState(singletonDB),
		baseDB:         baseDB,
	}
}

func (s *state) Commit() error {
	return s.baseDB.Commit()
}

func (s *state) Close() error {
	// close underlying database
	return s.baseDB.Close()
}
