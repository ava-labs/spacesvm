// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var zeroAddress = (common.Address{})

func (t *TransactionContext) authorized(owner common.Address) bool {
	return bytes.Equal(owner[:], t.Sender[:])
}

func verifySpace(s string, t *TransactionContext) (*SpaceInfo, error) {
	i, has, err := GetSpaceInfo(t.Database, []byte(s))
	if err != nil {
		return nil, err
	}
	// Cannot set key if space doesn't exist
	if !has {
		return nil, ErrSpaceMissing
	}
	// Space cannot be updated if not owned by modifier
	if !t.authorized(i.Owner) {
		return nil, ErrUnauthorized
	}
	// Space cannot be updated if expired
	//
	// This should never happen as expired records should be removed before
	// execution.
	if i.Expiry < t.BlockTime {
		return nil, ErrSpaceExpired
	}
	return i, nil
}

func updateSpace(s string, t *TransactionContext, timeRemaining uint64, i *SpaceInfo) error {
	newTimeRemaining := timeRemaining / i.Units
	i.Updated = t.BlockTime
	lastExpiry := i.Expiry
	i.Expiry = t.BlockTime + newTimeRemaining
	return PutSpaceInfo(t.Database, []byte(s), i, lastExpiry)
}

func valueUnits(g *Genesis, size uint64) uint64 {
	return size/g.ValueUnitSize + 1
}

func valueHash(v []byte) string {
	h := common.BytesToHash(crypto.Keccak256(v)).Hex()
	return strings.ToLower(h)
}
