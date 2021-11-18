// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package quarkvm

import (
	"errors"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/utils/hashing"
)

var (
	errTimestampTooEarly = errors.New("block's timestamp is earlier than its parent's timestamp")
	errDatabaseGet       = errors.New("error while retrieving data from database")
	errTimestampTooLate  = errors.New("block's timestamp is more than 1 hour ahead of local time")
	errBlockType         = errors.New("unexpected block type")

	_ Block = &timeBlock{}
)

type Block interface {
	snowman.Block
	Initialize(bytes []byte, status choices.Status, vm *VM)
	Data() [dataLen]byte
}

// Block is a block on the chain.
// Each block contains:
// 1) A piece of data (a string)
// 2) A timestamp
type timeBlock struct {
	PrntID ids.ID        `serialize:"true" json:"parentID"`  // parent's ID
	Hght   uint64        `serialize:"true" json:"height"`    // This block's height. The genesis block is at height 0.
	Tmstmp int64         `serialize:"true" json:"timestamp"` // Time this block was proposed at
	Dt     [dataLen]byte `serialize:"true" json:"data"`      // Arbitrary data

	id     ids.ID
	bytes  []byte
	status choices.Status
	vm     *VM
}

// Verify returns nil iff this block is valid.
// To be valid, it must be that:
// b.parent.Timestamp < b.Timestamp <= [local time] + 1 hour
func (b *timeBlock) Verify() error {
	// TODO: need to use versionDB (on accept/reject state)
	if b.Status() == choices.Accepted {
		return nil
	}

	// Get [b]'s parent
	parentID := b.Parent()
	parentIntf, err := b.vm.GetBlock(parentID)
	if err != nil {
		return errDatabaseGet
	}
	parent, ok := parentIntf.(*timeBlock)
	if !ok {
		return errBlockType
	}

	// Ensure [b]'s timestamp is after its parent's timestamp.
	if b.Timestamp().Unix() < parent.Timestamp().Unix() {
		return errTimestampTooEarly
	}

	// Ensure [b]'s timestamp is not more than an hour
	// ahead of this node's time
	if b.Timestamp().Unix() >= time.Now().Add(time.Hour).Unix() {
		return errTimestampTooLate
	}

	b.vm.currentBlocks[b.id] = b

	return nil
}

// Initialize sets [b.bytes] to [bytes], sets [b.id] to hash([b.bytes])
// Checks if [b]'s status is already stored in state. If so, [b] gets that status.
// Otherwise [b]'s status is Unknown.
func (b *timeBlock) Initialize(bytes []byte, status choices.Status, vm *VM) {
	b.vm = vm
	b.bytes = bytes
	b.id = hashing.ComputeHash256Array(b.bytes)
	b.status = status
}

// Accept sets this block's status to Accepted and sets lastAccepted to this
// block's ID and saves this info to b.vm.DB
func (b *timeBlock) Accept() error {
	b.SetStatus(choices.Accepted) // Change state of this block
	blkID := b.ID()

	// Persist data
	if err := b.vm.state.PutBlock(b); err != nil {
		return err
	}

	b.vm.state.SetLastAccepted(blkID) // Change state of VM
	if err := b.vm.state.Commit(); err != nil {
		return err
	}
	delete(b.vm.currentBlocks, b.ID())
	return nil
}

// Reject sets this block's status to Rejected and saves the status in state
// Recall that b.vm.DB.Commit() must be called to persist to the DB
func (b *timeBlock) Reject() error {
	b.SetStatus(choices.Rejected)
	if err := b.vm.state.PutBlock(b); err != nil {
		return err
	}
	if err := b.vm.state.Commit(); err != nil {
		return err
	}
	delete(b.vm.currentBlocks, b.ID())
	return nil
}

// ID returns the ID of this block
func (b *timeBlock) ID() ids.ID { return b.id }

// ParentID returns [b]'s parent's ID
func (b *timeBlock) Parent() ids.ID { return b.PrntID }

// Height returns this block's height. The genesis block has height 0.
func (b *timeBlock) Height() uint64 { return b.Hght }

// Timestamp returns this block's time. The genesis block has time 0.
func (b *timeBlock) Timestamp() time.Time { return time.Unix(b.Tmstmp, 0) }

// Status returns the status of this block
func (b *timeBlock) Status() choices.Status { return b.status }

// Bytes returns the byte repr. of this block
func (b *timeBlock) Bytes() []byte { return b.bytes }

// Data returns the data of this block
func (b *timeBlock) Data() [dataLen]byte { return b.Dt }

// SetStatus sets the status of this block
func (b *timeBlock) SetStatus(status choices.Status) { b.status = status }

func newTimeBlock(parentID ids.ID, height uint64, data [dataLen]byte, timestamp time.Time) *timeBlock {
	// Create our new block
	return &timeBlock{
		PrntID: parentID,
		Hght:   height,
		Tmstmp: timestamp.Unix(),
		Dt:     data,
	}
}
