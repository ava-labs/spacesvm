package chain

import (
	"time"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/quarkvm/types"
)

// TODO: load from genesis
const (
	lookbackWindow     = 10
	blockTarget        = 1
	targetTransactions = 10 * lookbackWindow / blockTarget // TODO: can be higher on real network
	blockTimer         = 250 * time.Millisecond            // TODO: set to be block target on real network
	minDifficulty      = 1                                 // TODO: set much higher on real network
	minBlockCost       = 0                                 // in units of tx surplus
	mempoolSize        = 1024
	maxKeyLength       = 256
	expiryTime         = 30 // TODO: set much longer on real network
)

type VM interface {
	State() DB
	Get(ids.ID) (*Block, error)
	Recents(currentTime int64, parent *Block) (recentBlockIDs ids.Set, recentTxIDs ids.Set, cost uint64, difficulty uint64)

	Verified(*Block) error
	Rejected(*Block) error
	Accepted(*Block) error
}

type DB interface {
	HasPrefix([]byte) (bool, error)
	HasPrefixKey([]byte, []byte) (bool, error)
	GetPrefixInfo([]byte) (*types.PrefixInfo, error)
	GetPrefixKey([]byte, []byte) ([]byte, error)
	PutPrefixInfo([]byte, *types.PrefixInfo) error
	PutPrefixKey([]byte, []byte, []byte) error
	DeletePrefixKey([]byte, []byte) error
	DeleteAllPrefixKeys([]byte) error

	StoreTransaction(*Transaction) error
	SetLastAccepted(*Block) error

	Commit() error
	SetDatabase(DB)
}
