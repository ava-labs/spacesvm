package chain

import (
	"errors"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/utils/hashing"

	"github.com/ava-labs/quarkvm/codec"
)

func init() {
	codec.RegisterType(&Block{})
}

var _ snowman.Block = &Block{}

var (
	ErrTimestampTooEarly   = errors.New("block timestamp too early")
	ErrTimestampTooLate    = errors.New("block timestamp too late")
	ErrNoTxs               = errors.New("no transactions")
	ErrInvalidCost         = errors.New("invalid block cost")
	ErrInvalidDifficulty   = errors.New("invalid difficulty")
	ErrInsufficientSurplus = errors.New("insufficient surplus difficulty")
)

type Block struct {
	Prnt       ids.ID         `serialize:"true" json:"parent"`
	Tmstmp     int64          `serialize:"true" json:"timestamp"`
	Hght       uint64         `serialize:"true" json:"height"`
	Difficulty uint64         `serialize:"true" json:"difficulty"`
	Cost       uint64         `serialize:"true" json:"cost"`
	Txs        []*Transaction `serialize:"true" json:"txs"`

	raw []byte
	id  ids.ID
	st  choices.Status

	vm         VM
	children   []*Block
	onAcceptDB DB
}

func (b *Block) Initialize(
	source []byte,
	status choices.Status,
	vm VM,
) {
	id, err := ids.ToID(hashing.ComputeHash256(source))
	if err != nil {
		panic(err)
	}
	b.raw = source
	b.id = id
	b.st = status
	b.vm = vm
}

// implements "snowman.Block.choices.Decidable"
func (b *Block) ID() ids.ID { return b.id }

// implements "snowman.Block"
func (b *Block) Verify() error {
	if b.st == choices.Accepted {
		return nil
	}

	parentBlock, err := b.vm.Get(b.Prnt)
	if err != nil {
		return err
	}
	if len(b.Txs) == 0 {
		return ErrNoTxs
	}
	if b.Timestamp().Unix() < parentBlock.Timestamp().Unix() {
		return ErrTimestampTooEarly
	}
	if b.Timestamp().Unix() >= time.Now().Add(10*time.Second).Unix() {
		return ErrTimestampTooLate
	}
	recentBlockIDs, recentTxIDs, cost, difficulty := b.vm.Recents(b.Tmstmp, parentBlock)
	if b.Cost != cost {
		return ErrInvalidCost
	}
	if b.Difficulty != difficulty {
		return ErrInvalidDifficulty
	}
	var surplusDifficulty uint64
	for _, tx := range b.Txs {
		if err := tx.Verify(b.onAcceptDB, b.Tmstmp, recentBlockIDs, recentTxIDs, difficulty); err != nil {
			return err
		}
		surplusDifficulty += tx.Difficulty() - difficulty
	}

	// Ensure enough work is performed to compensate for block production speed
	if surplusDifficulty < difficulty*b.Cost {
		return ErrInsufficientSurplus
	}

	// Set last accepted block and store
	if err := b.onAcceptDB.SetLastAccepted(b); err != nil {
		return err
	}

	parentBlock.addChild(b)
	// TODO: set prefered
	return b.vm.Verified(b)
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
	return b.vm.Accepted(b)
}

// implements "snowman.Block.choices.Decidable"
func (b *Block) Reject() error {
	b.st = choices.Rejected
	return b.vm.Rejected(b)
}

// implements "snowman.Block.choices.Decidable"
func (b *Block) Status() choices.Status {
	return b.st
}

// implements "snowman.Block"
func (b *Block) Parent() ids.ID { return b.Prnt }

// implements "snowman.Block"
func (b *Block) Bytes() []byte { return b.raw }

// implements "snowman.Block"
func (b *Block) Height() uint64 {
	return b.Hght
}

// implements "snowman.Block"
func (b *Block) Timestamp() time.Time {
	return time.Unix(b.Tmstmp, 0)
}

func (b *Block) onAccept() (DB, error) {
	if b.st == choices.Accepted {
		return b.vm.State(), nil
	}
	if b.onAcceptDB == nil {
		parentBlock, err := b.vm.Get(b.Prnt)
		if err != nil {
			return nil, err
		}
		return parentBlock.onAccept()
	}
	return b.onAcceptDB, nil
}

func (b *Block) addChild(c *Block) {
	b.children = append(b.children, c)
}
