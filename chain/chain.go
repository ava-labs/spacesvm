package chain

import (
	"time"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
)

// TODO: load from genesis
const (
	MaxPrefixSize      = 256
	maxKeyLength       = 256
	maxValueLength     = 256
	maxGraffitiSize    = 256
	LookbackWindow     = 10
	BlockTarget        = 1
	TargetTransactions = 10 * LookbackWindow / BlockTarget // TODO: can be higher on real network
	blockTimer         = 250 * time.Millisecond            // TODO: set to be block target on real network
	MinDifficulty      = 1                                 // TODO: set much higher on real network
	MinBlockCost       = 0                                 // in units of tx surplus
	mempoolSize        = 1024
	expiryTime         = 30 // TODO: set much longer on real network
)

type Context struct {
	RecentBlockIDs ids.Set
	RecentTxIDs    ids.Set

	NextCost       uint64
	NextDifficulty uint64
}

type Mempool interface {
	Len() int
	Prune(ids.Set)
	PopMax() (*Transaction, uint64)
	Add(*Transaction) bool
}

type VM interface {
	State() database.Database
	Mempool() Mempool

	GetBlock(ids.ID) (snowman.Block, error)
	ExecutionContext(currentTime int64, parent *Block) (*Context, error)

	Verified(*Block)
	Rejected(*Block)
	Accepted(*Block)
}
