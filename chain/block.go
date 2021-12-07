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

var _ snowman.Block = &Block{}

type Block struct {
	Prnt       ids.ID         `serialize:"true" json:"parent"`
	Tmstmp     int64          `serialize:"true" json:"timestamp"`
	Hght       uint64         `serialize:"true" json:"height"`
	Difficulty uint64         `serialize:"true" json:"difficulty"`
	Cost       uint64         `serialize:"true" json:"cost"`
	Txs        []*Transaction `serialize:"true" json:"txs"`

	id          ids.ID
	st          choices.Status
	parentBlock *Block

	vm         VM
	children   []*Block
	onAcceptDB *versiondb.Database
}

func NewBlock(vm VM, parent *Block, tmstp int64, difficulty uint64, cost uint64) *Block {
	return &Block{
		Tmstmp:     tmstp,
		Prnt:       parent.ID(),
		Hght:       parent.Height() + 1,
		Difficulty: difficulty,
		Cost:       cost,

		vm:          vm,
		st:          choices.Processing,
		parentBlock: parent,
	}
}

func InitializeBlock(
	source []byte,
	status choices.Status,
	vm VM,
) (*Block, error) {
	b := new(Block)
	if _, err := Unmarshal(source, b); err != nil {
		return nil, err
	}
	b.st = status
	b.vm = vm
	return b, nil
}

// implements "snowman.Block.choices.Decidable"
func (b *Block) ID() ids.ID {
	if b.id == (ids.ID{}) {
		id, err := ids.ToID(hashing.ComputeHash256(b.Bytes()))
		if err != nil {
			panic(err)
		}
		b.id = id
	}
	return b.id
}

// TODO: remove this, very ugly
func (b *Block) SetVM(vm VM) {
	b.vm = vm
}

// implements "snowman.Block"
func (b *Block) Verify() error {
	if b.st == choices.Accepted {
		log.Debug("block already accepted", "id", b.ID())
		return nil
	}
	if b.parentBlock == nil {
		parentBlock, err := b.vm.Get(b.Prnt)
		if err != nil {
			log.Debug("could not get parent", "id", b.Prnt)
			return err
		}
		b.parentBlock = parentBlock
	}
	if len(b.Txs) == 0 {
		return ErrNoTxs
	}
	if b.Timestamp().Unix() < b.parentBlock.Timestamp().Unix() {
		return ErrTimestampTooEarly
	}
	// TODO: make future time bound a const
	if b.Timestamp().Unix() >= time.Now().Add(10*time.Second).Unix() {
		return ErrTimestampTooLate
	}
	recentBlockIDs, recentTxIDs, cost, difficulty := b.vm.Recents(b.Tmstmp, b.parentBlock)
	if b.Cost != cost {
		return ErrInvalidCost
	}
	if b.Difficulty != difficulty {
		return ErrInvalidDifficulty
	}
	parentState, err := b.parentBlock.OnAccept()
	if err != nil {
		return err
	}
	if b.onAcceptDB != nil {
		// Could happen if previously verified
		// TODO: probably a MUCH MUCH easier way to do this
		if err := b.onAcceptDB.Close(); err != nil {
			return err
		}
	}
	b.onAcceptDB = versiondb.New(parentState)
	var surplusDifficulty uint64
	for _, tx := range b.Txs {
		if err := tx.Verify(b.onAcceptDB, b.Tmstmp, recentBlockIDs, recentTxIDs, difficulty); err != nil {
			log.Error("failed tx verification", "err", err)
			return err
		}
		surplusDifficulty += tx.Difficulty() - difficulty
	}

	// Ensure enough work is performed to compensate for block production speed
	if surplusDifficulty < difficulty*b.Cost {
		return ErrInsufficientSurplus
	}

	// Set last accepted block and store
	if err := SetLastAccepted(b.onAcceptDB, b); err != nil {
		return err
	}

	b.parentBlock.addChild(b)
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
func (b *Block) Status() choices.Status { return b.st }

// implements "snowman.Block"
func (b *Block) Parent() ids.ID { return b.Prnt }

// implements "snowman.Block"
func (b *Block) Bytes() []byte {
	by, err := Marshal(b)
	if err != nil {
		panic(err)
	}
	return by
}

// implements "snowman.Block"
func (b *Block) Height() uint64 {
	return b.Hght
}

// implements "snowman.Block"
func (b *Block) Timestamp() time.Time {
	return time.Unix(b.Tmstmp, 0)
}

// TODO: make private when move production into chain package
func (b *Block) OnAccept() (database.Database, error) {
	if b.st == choices.Accepted || b.ID() == (ids.ID{}) /* genesis */ {
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
