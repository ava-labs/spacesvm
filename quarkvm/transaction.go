// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package quarkvm

import "github.com/ava-labs/avalanchego/ids"

type transaction interface {
	ID() ids.ID
	Difficulty() uint64
	GetBlockID() ids.ID
	Bytes() [32]byte
}

func newTransaction(d [32]byte) transaction {
	return &tx{}
}

type tx struct {
	// TODO
}

func (tx *tx) ID() ids.ID {
	return ids.Empty
}

func (tx *tx) Difficulty() uint64 {
	return 0
}

func (tx *tx) GetBlockID() ids.ID {
	return ids.Empty
}

func (tx *tx) Bytes() [32]byte {
	return ids.Empty
}
