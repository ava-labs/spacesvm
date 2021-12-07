package chain

import (
	"time"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
)

// TODO: load from genesis
const (
	MaxPrefixSize      = 256
	maxKeyLength       = 256
	maxValueLength     = 256
	LookbackWindow     = 10
	BlockTarget        = 1
	TargetTransactions = 10 * LookbackWindow / BlockTarget // TODO: can be higher on real network
	blockTimer         = 250 * time.Millisecond            // TODO: set to be block target on real network
	MinDifficulty      = 1                                 // TODO: set much higher on real network
	MinBlockCost       = 0                                 // in units of tx surplus
	mempoolSize        = 1024
	expiryTime         = 30 // TODO: set much longer on real network
)

type VM interface {
	State() database.Database
	// TODO: change naming
	Get(ids.ID) (*Block, error)
	Recents(currentTime int64, parent *Block) (recentBlockIDs ids.Set, recentTxIDs ids.Set, cost uint64, difficulty uint64)

	// TODO: change naming
	Verified(*Block) error
	Rejected(*Block) error
	Accepted(*Block) error
}
