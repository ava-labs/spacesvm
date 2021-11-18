package chain

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/quarkvm/block"
	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/owner"
	"github.com/ava-labs/quarkvm/storage"
	"github.com/ava-labs/quarkvm/transaction"
)

type Chain interface {
	GetBlock(id ids.ID) (*block.Block, error)
	AddBlock(b *block.Block)
	CurrentBlock() *block.Block
	SetLastAccepted(id ids.ID)
	GetLastAccepted() ids.ID
	PutBlock(*block.Block) error

	// TODO: separate mempool interface?
	Produce() *block.Block
	Submit(*transaction.Transaction)
	ValidBlockID(ids.ID) bool
	GetPrefixInfo(prefix []byte) (*owner.Owner, bool, error)
	GetValue(key []byte) ([]byte, bool, error)
	Pending() int
	MempoolContains(ids.ID) bool
	TxConfirmed(ids.ID) bool
	DifficultyEstimate() uint64
}

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

var _ Chain = &chain{}

type chain struct {
	mu sync.RWMutex

	s storage.Storage

	mempool transaction.Mempool

	// add when a block is produced
	// TODO: cache blocks
	blocks       []*block.Block
	lastAccepted ids.ID
}

func New(s storage.Storage) Chain {
	return &chain{
		s:       s,
		mempool: transaction.NewMempool(mempoolSize),
		blocks: []*block.Block{
			// for initial produce
			{
				MinDifficulty: minDifficulty,
				BlockCost:     minBlockCost,
			},
		},
	}
}

func (c *chain) GetBlock(id ids.ID) (*block.Block, error) {
	// TODO: read from cache
	blkBytes, err := c.s.Block().Get(id[:])
	if err != nil {
		return nil, err
	}

	blk := new(block.Block)
	if _, err := codec.Unmarshal(blkBytes, blk); err != nil {
		return nil, err
	}
	return blk, nil
}

func (c *chain) AddBlock(b *block.Block) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.blocks = append(c.blocks, b)
}

func (c *chain) CurrentBlock() *block.Block {
	return c.blocks[len(c.blocks)-1]
}

func (c *chain) SetLastAccepted(id ids.ID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastAccepted = id
}

func (c *chain) GetLastAccepted() ids.ID {
	return c.lastAccepted
}

func (c *chain) PutBlock(b *block.Block) error {
	id := b.ID()
	d, err := codec.Marshal(b)
	if err != nil {
		return err
	}
	return c.s.Block().Put(id[:], d)
}

func (c *chain) ValidBlockID(blockID ids.ID) bool {
	now := time.Now().Unix()
	for i := len(c.blocks) - 1; i >= 0; i-- {
		ob := c.blocks[i]
		sinceBlock := now - ob.Timestamp().Unix()
		if sinceBlock > lookbackWindow && i != len(c.blocks)-1 {
			break
		}
		if ob.ID() == blockID {
			return true
		}
	}
	return false
}

func (c *chain) Submit(tx *transaction.Transaction) {
	// Important to cache the difficulty of the transaction prior to block
	// production otherwise the block prod loop will be SUPER slow.
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mempool.Push(tx)
}

func (c *chain) GetPrefixInfo(prefix []byte) (*owner.Owner, bool, error) {
	has, err := c.s.Owner().Has(prefix)
	if err != nil {
		return nil, false, err
	}
	if !has {
		return nil, false, nil
	}
	v, err := c.s.Owner().Get(prefix)
	if err != nil {
		return nil, false, err
	}
	iv := new(owner.Owner)
	if _, err := codec.Unmarshal(v, iv); err != nil {
		return nil, false, err
	}
	return iv, true, nil
}

func (c *chain) GetValue(key []byte) ([]byte, bool, error) {
	return c.s.Get(key)
}

func (c *chain) DifficultyEstimate() uint64 {
	totalDifficulty := uint64(0)
	totalBlocks := uint64(0)
	currTime := time.Now().Unix()
	for i := len(c.blocks) - 1; i >= 0; i-- {
		ob := c.blocks[i]
		sinceBlock := currTime - ob.Timestamp().Unix()
		if sinceBlock > lookbackWindow/2 && i != len(c.blocks)-1 {
			break
		}
		totalDifficulty += ob.MinDifficulty
		totalBlocks++
	}
	return totalDifficulty/totalBlocks + 1
}

func (c *chain) RecentData(currTime int64, lastBlock *block.Block) (ids.Set, ids.Set, uint64, uint64) {
	// get tx count over lookback
	recentBlockIDs := ids.NewSet(256) // TODO: figure out right sizes here, keep track of dynamically
	recentTxIDs := ids.NewSet(256)
	for i := len(c.blocks) - 1; i >= 0; i-- {
		ob := c.blocks[i]
		sinceBlock := currTime - ob.Timestamp().Unix()
		if sinceBlock > lookbackWindow && i != len(c.blocks)-1 {
			break
		}
		recentBlockIDs.Add(ob.ID())
		for _, tx := range ob.Txs {
			recentTxIDs.Add(tx.ID())
		}
	}

	// compute new block cost
	secondsSinceLast := currTime - lastBlock.Timestamp().Unix()
	newBlockCost := lastBlock.BlockCost
	if secondsSinceLast < blockTarget {
		newBlockCost += uint64(blockTarget - secondsSinceLast)
	} else {
		possibleDiff := uint64(secondsSinceLast - blockTarget)
		if possibleDiff < newBlockCost-minBlockCost {
			newBlockCost -= possibleDiff
		} else {
			newBlockCost = minBlockCost
		}
	}

	// compute new min difficulty
	newMinDifficulty := lastBlock.MinDifficulty
	recentTxs := recentTxIDs.Len()
	if recentTxs > targetTransactions {
		newMinDifficulty++
	} else if recentTxs < targetTransactions {
		elapsedWindows := uint64(secondsSinceLast/lookbackWindow) + 1 // account for current window being less
		if elapsedWindows < newMinDifficulty-minDifficulty {
			newMinDifficulty -= elapsedWindows
		} else {
			newMinDifficulty = minDifficulty
		}
	}

	return recentBlockIDs, recentTxIDs, newBlockCost, newMinDifficulty
}

func (c *chain) Produce() *block.Block {
	lb := c.blocks[len(c.blocks)-1]
	now := time.Now().Unix()
	recentBlockIDs, recentTxIDs, blockCost, minDifficulty := c.RecentData(now, lb)

	// TODO: should be from ParseBlock
	b := &block.Block{
		Tmstmp:        now,
		Prnt:          lb.ID(),
		BlockCost:     blockCost,
		MinDifficulty: minDifficulty,
	}

	// select new transactions
	c.mu.Lock()
	defer c.mu.Unlock()
	b.Txs = []*transaction.Transaction{}
	c.mempool.Prune(recentBlockIDs)
	prefixes := ids.NewSet(targetTransactions)
	for len(b.Txs) < targetTransactions && c.mempool.Len() > 0 {
		next, diff := c.mempool.PopMax()
		if diff < b.MinDifficulty {
			c.mempool.Push(next)
			break
		}
		p := next.PrefixID()
		if prefixes.Contains(p) {
			continue
		}
		if err := next.Verify(c.s, b.Timestamp().Unix(), recentBlockIDs, recentTxIDs, b.MinDifficulty); err != nil {
			fmt.Println("dropping tx", "id:", next.ID(), "err:", err)
			continue
		}
		// Wait to add prefix until after verification
		prefixes.Add(p)
		b.Txs = append(b.Txs, next)
	}

	return b
}

func (c *chain) Pending() int {
	c.mu.RLock()
	cnt := c.mempool.Len()
	c.mu.RUnlock()
	return cnt
}

func (c *chain) MempoolContains(txID ids.ID) bool {
	c.mu.RLock()
	has := c.mempool.Has(txID)
	c.mu.RUnlock()
	return has
}

func (c *chain) TxConfirmed(txID ids.ID) bool {
	id := append([]byte{}, txID[:]...)
	has, err := c.s.Tx().Has(id)
	if err != nil {
		panic(err)
	}
	return has
}

func (c *chain) Verify(b *block.Block) error {
	if len(b.Txs) == 0 {
		return errors.New("no fee-paying transactions")
	}
	lastBlock := c.blocks[len(c.blocks)-1]
	if b.Timestamp().Unix() < lastBlock.Timestamp().Unix() {
		return errors.New("invalid block time")
	}
	recentBlockIDs, recentTxIDs, blockCost, minDifficulty := c.RecentData(b.Timestamp().Unix(), lastBlock)
	if b.BlockCost != blockCost {
		return errors.New("invalid block cost")
	}
	if b.MinDifficulty != minDifficulty {
		return errors.New("invalid difficulty")
	}
	// Ensure only 1 claim per prefix per block (otherwise both may pass Verify
	// and one will fail on accept)
	prefixes := ids.NewSet(len(b.Txs))
	surplusDifficulty := uint64(0)
	for _, tx := range b.Txs {
		p := tx.PrefixID()
		if prefixes.Contains(p) {
			return errors.New("only 1 operation per prefix allowed per block")
		}
		prefixes.Add(p)
		if err := tx.Verify(c.s, b.Timestamp().Unix(), recentBlockIDs, recentTxIDs, minDifficulty); err != nil {
			return err
		}
		surplusDifficulty += tx.Difficulty() - minDifficulty
	}

	// Ensure enough work is performed to compensate for block production speed
	if surplusDifficulty < minDifficulty*b.BlockCost {
		return errors.New("insufficient block burn")
	}
	return nil
}
