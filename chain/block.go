// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	log "github.com/inconshreveable/log15"
	"golang.org/x/crypto/sha3"
)

const futureBound = 10 * time.Second

var _ snowman.Block = &StatelessBlock{}

type StatefulBlock struct {
	Prnt   ids.ID `serialize:"true" json:"parent"`
	Tmstmp int64  `serialize:"true" json:"timestamp"`
	// e.g., encodes extra data for genesis
	ExtraData  []byte         `serialize:"true" json:"extraData"`
	Hght       uint64         `serialize:"true" json:"height"`
	Difficulty uint64         `serialize:"true" json:"difficulty"`
	Cost       uint64         `serialize:"true" json:"cost"`
	Txs        []*Transaction `serialize:"true" json:"txs"`
}

// Stateless is defined separately from "Block"
// in case external packages needs use the stateful block
// without mocking VM or parent block
type StatelessBlock struct {
	*StatefulBlock `serialize:"true" json:"block"`

	id    ids.ID
	st    choices.Status
	t     time.Time
	bytes []byte

	vm         VM
	children   []*StatelessBlock
	onAcceptDB *versiondb.Database
}

func NewBlock(vm VM, parent snowman.Block, tmstp int64, context *Context) *StatelessBlock {
	return &StatelessBlock{
		StatefulBlock: &StatefulBlock{
			Tmstmp:     tmstp,
			Prnt:       parent.ID(),
			Hght:       parent.Height() + 1,
			Difficulty: context.NextDifficulty,
			Cost:       context.NextCost,
		},
		vm: vm,
		st: choices.Processing,
	}
}

// TODO: check work here? Seems like a DoS vuln?
func ParseBlock(
	source []byte,
	status choices.Status,
	vm VM,
) (*StatelessBlock, error) {
	blk := new(StatefulBlock)
	if _, err := Unmarshal(source, blk); err != nil {
		return nil, err
	}
	b := &StatelessBlock{
		StatefulBlock: blk,
		t:             time.Unix(blk.Tmstmp, 0),
		bytes:         source,
		st:            status,
		vm:            vm,
	}
	h := sha3.Sum256(b.bytes)
	id, err := ids.ToID(h[:])
	if err != nil {
		return nil, err
	}
	b.id = id
	for _, tx := range blk.Txs {
		if err := tx.Init(); err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (b *StatelessBlock) init() error {
	bytes, err := Marshal(b.StatefulBlock)
	if err != nil {
		return err
	}
	b.bytes = bytes

	h := sha3.Sum256(b.bytes)
	id, err := ids.ToID(h[:])
	if err != nil {
		return err
	}
	b.id = id
	b.t = time.Unix(b.StatefulBlock.Tmstmp, 0)
	for _, tx := range b.StatefulBlock.Txs {
		if err := tx.Init(); err != nil {
			return err
		}
	}
	return nil
}

// implements "snowman.Block.choices.Decidable"
func (b *StatelessBlock) ID() ids.ID { return b.id }

// verify checks the correctness of a block and then returns the
// *versiondb.Database computed during execution.
func (b *StatelessBlock) verify() (*StatelessBlock, *versiondb.Database, error) {
	prnt, err := b.vm.GetBlock(b.Prnt)
	if err != nil {
		log.Debug("could not get parent", "id", b.Prnt)
		return nil, nil, err
	}
	parent, ok := prnt.(*StatelessBlock)
	if !ok {
		return nil, nil, fmt.Errorf("unexpected snowman.Block %T, expected *StatelessBlock", prnt)
	}

	if len(b.Txs) == 0 {
		return nil, nil, ErrNoTxs
	}
	if b.Timestamp().Unix() < parent.Timestamp().Unix() {
		return nil, nil, ErrTimestampTooEarly
	}
	if b.Timestamp().Unix() >= time.Now().Add(futureBound).Unix() {
		return nil, nil, ErrTimestampTooLate
	}
	context, err := b.vm.ExecutionContext(b.Tmstmp, parent)
	if err != nil {
		return nil, nil, err
	}
	if b.Cost != context.NextCost {
		return nil, nil, ErrInvalidCost
	}
	if b.Difficulty != context.NextDifficulty {
		return nil, nil, ErrInvalidDifficulty
	}
	parentState, err := parent.onAccept()
	if err != nil {
		return nil, nil, err
	}
	onAcceptDB := versiondb.New(parentState)

	// Remove all expired prefixes
	if err := ExpireNext(onAcceptDB, parent.Tmstmp, b.Tmstmp); err != nil {
		return nil, nil, err
	}

	// Process new transactions
	log.Debug("build context", "next difficulty", context.NextDifficulty, "next cost", context.NextCost)
	var surplusDifficulty uint64
	for _, tx := range b.Txs {
		if err := tx.Execute(onAcceptDB, b.Tmstmp, context); err != nil {
			log.Debug("failed tx verification", "err", err)
			return nil, nil, err
		}
		surplusDifficulty += tx.Difficulty() - context.NextDifficulty
	}
	// Ensure enough work is performed to compensate for block production speed
	requiredSurplus := b.Difficulty * b.Cost
	if surplusDifficulty < requiredSurplus {
		log.Debug("insufficient block surplus", "found", surplusDifficulty, "required", requiredSurplus)
		return nil, nil, ErrInsufficientSurplus
	}
	return parent, onAcceptDB, nil
}

// implements "snowman.Block"
func (b *StatelessBlock) Verify() error {
	parent, onAcceptDB, err := b.verify()
	if err != nil {
		return err
	}
	b.onAcceptDB = onAcceptDB

	// Set last accepted block and store
	if err := SetLastAccepted(b.onAcceptDB, b); err != nil {
		return err
	}

	parent.addChild(b)
	b.vm.Verified(b)
	return nil
}

// implements "snowman.Block.choices.Decidable"
func (b *StatelessBlock) Accept() error {
	if err := b.onAcceptDB.Commit(); err != nil {
		return err
	}
	for _, child := range b.children {
		if err := child.onAcceptDB.SetDatabase(b.vm.State()); err != nil {
			return err
		}
	}
	b.st = choices.Accepted
	b.vm.Accepted(b)
	return nil
}

// implements "snowman.Block.choices.Decidable"
func (b *StatelessBlock) Reject() error {
	b.st = choices.Rejected
	b.vm.Rejected(b)
	return nil
}

// implements "snowman.Block.choices.Decidable"
func (b *StatelessBlock) Status() choices.Status { return b.st }

// implements "snowman.Block"
func (b *StatelessBlock) Parent() ids.ID { return b.StatefulBlock.Prnt }

// implements "snowman.Block"
func (b *StatelessBlock) Bytes() []byte { return b.bytes }

// implements "snowman.Block"
func (b *StatelessBlock) Height() uint64 { return b.StatefulBlock.Hght }

// implements "snowman.Block"
func (b *StatelessBlock) Timestamp() time.Time { return b.t }

func (b *StatelessBlock) SetChildrenDB(db database.Database) error {
	for _, child := range b.children {
		if err := child.onAcceptDB.SetDatabase(db); err != nil {
			return err
		}
	}
	return nil
}

func (b *StatelessBlock) onAccept() (database.Database, error) {
	if b.st == choices.Accepted || b.Hght == 0 /* genesis */ {
		return b.vm.State(), nil
	}
	if b.onAcceptDB != nil {
		return b.onAcceptDB, nil
	}
	return nil, ErrParentBlockNotVerified
}

func (b *StatelessBlock) addChild(c *StatelessBlock) {
	b.children = append(b.children, c)
}
