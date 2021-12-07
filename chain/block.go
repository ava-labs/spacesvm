package chain

import (
	"time"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/utils/hashing"
	log "github.com/inconshreveable/log15"
)

const (
	futureBound = 10 * time.Second
)

var _ snowman.Block = &Block{}

type Block struct {
	Prnt       ids.ID         `serialize:"true" json:"parent"`
	Tmstmp     int64          `serialize:"true" json:"timestamp"`
	Hght       uint64         `serialize:"true" json:"height"`
	Difficulty uint64         `serialize:"true" json:"difficulty"`
	Cost       uint64         `serialize:"true" json:"cost"`
	Txs        []*Transaction `serialize:"true" json:"txs"`

	id    ids.ID
	st    choices.Status
	t     time.Time
	bytes []byte

	vm         VM
	children   []*Block
	onAcceptDB *versiondb.Database
}

func NewBlock(vm VM, parent *Block, tmstp int64, context *Context) *Block {
	return &Block{
		Tmstmp:     tmstp,
		Prnt:       parent.ID(),
		Hght:       parent.Height() + 1,
		Difficulty: context.NextDifficulty,
		Cost:       context.NextCost,

		vm: vm,
		st: choices.Processing,
	}
}

// TODO: check work here? Seems like a DoS vuln?
func ParseBlock(
	source []byte,
	status choices.Status,
	vm VM,
) (*Block, error) {
	b := new(Block)
	b.bytes = source
	if _, err := Unmarshal(source, b); err != nil {
		return nil, err
	}
	id, err := ids.ToID(hashing.ComputeHash256(b.bytes))
	if err != nil {
		return nil, err
	}
	b.id = id
	b.t = time.Unix(b.Tmstmp, 0)
	b.st = status
	b.vm = vm
	return b, nil
}

func (b *Block) init() error {
	bytes, err := Marshal(b)
	if err != nil {
		return err
	}
	b.bytes = bytes
	id, err := ids.ToID(hashing.ComputeHash256(b.bytes))
	if err != nil {
		return err
	}
	b.id = id
	b.t = time.Unix(b.Tmstmp, 0)
	return nil
}

// implements "snowman.Block.choices.Decidable"
func (b *Block) ID() ids.ID { return b.id }

// verify checks the correctness of a block and then returns the
// *versiondb.Database computed during execution.
func (b *Block) verify() (*Block, *versiondb.Database, error) {
	prnt, err := b.vm.GetBlock(b.Prnt)
	if err != nil {
		log.Debug("could not get parent", "id", b.Prnt)
		return nil, nil, err
	}
	parent := prnt.(*Block)
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
	var surplusDifficulty uint64
	for _, tx := range b.Txs {
		if err := tx.Verify(onAcceptDB, b.Tmstmp, context); err != nil {
			log.Debug("failed tx verification", "err", err)
			return nil, nil, err
		}
		surplusDifficulty += tx.Difficulty() - context.NextDifficulty
	}
	// Ensure enough work is performed to compensate for block production speed
	if surplusDifficulty < b.Difficulty*b.Cost {
		return nil, nil, ErrInsufficientSurplus
	}
	return parent, onAcceptDB, nil
}

// implements "snowman.Block"
func (b *Block) Verify() error {
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
func (b *Block) Accept() error {
	if err := b.onAcceptDB.Commit(); err != nil {
		return err
	}
	for _, child := range b.children {
		child.onAcceptDB.SetDatabase(b.vm.State())
	}
	b.st = choices.Accepted
	b.vm.Accepted(b)
	return nil
}

// implements "snowman.Block.choices.Decidable"
func (b *Block) Reject() error {
	b.st = choices.Rejected
	b.vm.Rejected(b)
	return nil
}

// implements "snowman.Block.choices.Decidable"
func (b *Block) Status() choices.Status { return b.st }

// implements "snowman.Block"
func (b *Block) Parent() ids.ID { return b.Prnt }

// implements "snowman.Block"
func (b *Block) Bytes() []byte { return b.bytes }

// implements "snowman.Block"
func (b *Block) Height() uint64 { return b.Hght }

// implements "snowman.Block"
func (b *Block) Timestamp() time.Time { return b.t }

func (b *Block) onAccept() (database.Database, error) {
	if b.st == choices.Accepted || b.Hght == 0 /* genesis */ {
		return b.vm.State(), nil
	}
	if b.onAcceptDB != nil {
		return b.onAcceptDB, nil
	}
	return nil, ErrParentBlockNotVerified
}

func (b *Block) addChild(c *Block) {
	b.children = append(b.children, c)
}
