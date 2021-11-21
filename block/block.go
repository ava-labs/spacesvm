package block

import (
	"errors"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/storage"
	"github.com/ava-labs/quarkvm/transaction"
)

func init() {
	codec.RegisterType(&Block{})
}

var _ snowman.Block = &Block{}

var (
	ErrTimestampTooEarly = errors.New("block timestamp too early")
	ErrTimestampTooLate  = errors.New("block timestamp too late")
)

type Block struct {
	Prnt   ids.ID `serialize:"true" json:"parentID"`
	Tmstmp int64  `serialize:"true" json:"timestamp"`
	Hght   uint64 `serialize:"true" json:"height"`

	MinDifficulty uint64                     `serialize:"true" json:"minDifficulty"`
	BlockCost     uint64                     `serialize:"true" json:"blockCost"`
	Txs           []*transaction.Transaction `serialize:"true" json:"txs"`

	raw      []byte
	id       ids.ID
	st       choices.Status
	s        storage.Storage
	lookup   func(ids.ID) (*Block, error)
	onVerify func(*Block) error
	onAccept func(*Block) error
	onReject func(*Block) error
}

func (b *Block) Update(
	source []byte,
	status choices.Status,
	s storage.Storage,
	lookup func(ids.ID) (*Block, error),
	onVerify func(*Block) error,
	onAccept func(*Block) error,
	onReject func(*Block) error,
) {
	id, err := ids.ToID(hashing.ComputeHash256(source))
	if err != nil {
		panic(err)
	}
	b.raw = source
	b.id = id
	b.st = status
	b.s = s
	b.lookup = lookup
	b.onVerify = onVerify
	b.onAccept = onAccept
	b.onReject = onReject
}

// implements "snowman.Block.choices.Decidable"
func (b *Block) ID() ids.ID { return b.id }

// implements "snowman.Block"
func (b *Block) Verify() error {
	if b.st == choices.Accepted {
		return nil
	}

	parentID := b.Parent()
	parentBlock, err := b.lookup(parentID)
	if err != nil {
		return err
	}

	if b.Timestamp().Unix() < parentBlock.Timestamp().Unix() {
		return ErrTimestampTooEarly
	}
	if b.Timestamp().Unix() >= time.Now().Add(time.Hour).Unix() {
		return ErrTimestampTooLate
	}

	return b.onVerify(b)
}

// implements "snowman.Block.choices.Decidable"
func (b *Block) Accept() error {
	for _, tx := range b.Txs {
		if err := tx.Accept(b.s, b.Tmstmp); err != nil {
			return err
		}
	}
	b.st = choices.Accepted
	return b.onAccept(b)
}

// implements "snowman.Block.choices.Decidable"
func (b *Block) Reject() error {
	b.st = choices.Rejected
	return b.onReject(b)
}

// implements "snowman.Block.choices.Decidable"
func (b *Block) Status() choices.Status {
	return b.st
}

// implements "snowman.Block"
func (b *Block) Parent() ids.ID { return b.Prnt }

// implements "snowman.Block"
func (b *Block) Bytes() []byte {
	d, err := codec.Marshal(b)
	if err != nil {
		panic(err)
	}
	return d
}

// implements "snowman.Block"
func (b *Block) Height() uint64 {
	return b.Hght
}

// implements "snowman.Block"
func (b *Block) Timestamp() time.Time {
	return time.Unix(b.Tmstmp, 0)
}
