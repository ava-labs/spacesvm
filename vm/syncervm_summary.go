// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package vm implements custom VM.
package vm

import (
	"fmt"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/codec/linearcodec"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	Version        = uint16(0)
	maxMessageSize = 1 * units.MiB
)

var (
	_     block.StateSummary = &SyncSummary{}
	Codec codec.Manager
)

func init() {
	Codec = codec.NewManager(maxMessageSize)
	c := linearcodec.NewDefault()

	errs := wrappers.Errs{}
	errs.Add(
		// Types for state sync frontier consensus
		c.RegisterType(SyncSummary{}),

		Codec.RegisterCodec(Version, c),
	)

	if errs.Errored() {
		panic(errs.Err)
	}
}

// SyncSummary provides the information necessary to sync a node starting
// at the given block.
type SyncSummary struct {
	BlockNumber uint64 `serialize:"true"`
	BlockHash   ids.ID `serialize:"true"`
	BlockRoot   ids.ID `serialize:"true"`

	summaryID  ids.ID
	bytes      []byte
	acceptImpl func(SyncSummary) (bool, error)
}

func NewSyncSummaryFromBytes(summaryBytes []byte, acceptImpl func(SyncSummary) (bool, error)) (SyncSummary, error) {
	summary := SyncSummary{}
	if codecVersion, err := Codec.Unmarshal(summaryBytes, &summary); err != nil {
		return SyncSummary{}, err
	} else if codecVersion != Version {
		return SyncSummary{}, fmt.Errorf("failed to parse syncable summary due to unexpected codec version (%d != %d)", codecVersion, Version)
	}

	summary.bytes = summaryBytes
	summary.summaryID = hashing.ComputeHash256Array(summaryBytes)
	summary.acceptImpl = acceptImpl
	return summary, nil
}

func NewSyncSummary(blockHash ids.ID, blockNumber uint64, blockRoot ids.ID) (SyncSummary, error) {
	summary := SyncSummary{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
		BlockRoot:   blockRoot,
	}
	bytes, err := Codec.Marshal(Version, &summary)
	if err != nil {
		return SyncSummary{}, err
	}

	summary.bytes = bytes
	summaryID, err := ids.ToID(crypto.Keccak256(bytes))
	if err != nil {
		return SyncSummary{}, err
	}
	summary.summaryID = summaryID

	return summary, nil
}

func (s SyncSummary) Bytes() []byte {
	return s.bytes
}

func (s SyncSummary) Height() uint64 {
	return s.BlockNumber
}

func (s SyncSummary) ID() ids.ID {
	return s.summaryID
}

func (s SyncSummary) String() string {
	return fmt.Sprintf("SyncSummary(BlockHash=%s, BlockNumber=%d, BlockRoot=%s)", s.BlockHash, s.BlockNumber, s.BlockRoot)
}

func (s SyncSummary) Accept() (bool, error) {
	if s.acceptImpl == nil {
		return false, fmt.Errorf("accept implementation not specified for summary: %s", s)
	}
	return s.acceptImpl(s)
}
