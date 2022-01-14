// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/spacesvm/parser"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const hashLen = 64

var _ UnsignedTransaction = &SetTx{}

type SetTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`

	// Space is the namespace for the "PrefixInfo"
	// whose owner can write and read value for the
	// specific key space.
	// The space must be ^[a-z0-9]{1,256}$.
	Space string `serialize:"true" json:"space"`

	// Key is parsed from the given input, with its space removed.
	Key string `serialize:"true" json:"key"`

	// Value is writen as the key-value pair to the storage. If a previous value
	// exists, it is overwritten.
	Value []byte `serialize:"true" json:"value"`
}

func (s *SetTx) Execute(t *TransactionContext) error {
	g := t.Genesis
	if err := parser.CheckContents(s.Space); err != nil {
		return err
	}
	if err := parser.CheckContents(s.Key); err != nil {
		return err
	}
	switch {
	case len(s.Value) == 0:
		return ErrValueEmpty
	case uint64(len(s.Value)) > g.MaxValueSize:
		return ErrValueTooBig
	}

	// Verify space is owned by sender
	i, err := verifySpace(s.Space, t)
	if err != nil {
		return err
	}

	// If Key is equal to hash length, ensure it is equal to the hash of the
	// value
	if len(s.Key) == hashLen {
		h := common.BytesToHash(crypto.Keccak256(s.Value)).Hex()
		h = strings.ToLower(h)
		if s.Key != h {
			return fmt.Errorf("%w: expected %s got %x", ErrInvalidKey, h, s.Key)
		}
	}

	// Update value
	v, exists, err := GetValue(t.Database, []byte(s.Space), []byte(s.Key))
	if err != nil {
		return err
	}
	timeRemaining := (i.Expiry - i.LastUpdated) * i.Units
	if exists {
		i.Units -= valueUnits(g, v)
	}
	i.Units += valueUnits(g, s.Value)
	if err := PutSpaceKey(t.Database, []byte(s.Space), []byte(s.Key), t.TxID[:]); err != nil {
		return err
	}
	return updateSpace(s.Space, t, timeRemaining, i)
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
	if !bytes.Equal(i.Owner[:], t.Sender[:]) {
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
	i.LastUpdated = t.BlockTime
	lastExpiry := i.Expiry
	i.Expiry = t.BlockTime + newTimeRemaining
	return PutSpaceInfo(t.Database, []byte(s), i, lastExpiry)
}

func valueUnits(g *Genesis, b []byte) uint64 {
	return uint64(len(b))/g.ValueUnitSize + 1
}

func (s *SetTx) FeeUnits(g *Genesis) uint64 {
	// We don't subtract by 1 here because we want to charge extra for any
	// value-based interaction (even if it is small or a delete).
	return s.BaseTx.FeeUnits(g) + valueUnits(g, s.Value)
}

func (s *SetTx) LoadUnits(g *Genesis) uint64 {
	return s.FeeUnits(g)
}

func (s *SetTx) Copy() UnsignedTransaction {
	value := make([]byte, len(s.Value))
	copy(value, s.Value)
	return &SetTx{
		BaseTx: s.BaseTx.Copy(),
		Space:  s.Space,
		Key:    s.Key,
		Value:  value,
	}
}
