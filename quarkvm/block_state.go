// (c) 2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package timestampvm

import (
	"github.com/ava-labs/avalanchego/cache"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
)

const (
	blockCacheSize = 8192
)

var _ BlockState = &blockState{}

type BlockState interface {
	GetBlock(blkID ids.ID) (Block, error)
	PutBlock(blk Block) error

	GetLastAccepted() ids.ID
	SetLastAccepted(ids.ID)

	ClearCache()
}

type blockState struct {
	blkCache cache.Cacher
	blockDB  database.Database
	vm       *VM

	lastAccepted ids.ID
}

type blkWrapper struct {
	Blk    []byte         `serialize:"true"`
	Status choices.Status `serialize:"true"`

	block Block
}

func NewBlockState(db database.Database, vm *VM) BlockState {
	return &blockState{
		blkCache: &cache.LRU{Size: blockCacheSize},
		blockDB:  db,
		vm:       vm,
	}
}

func (s *blockState) GetBlock(blkID ids.ID) (Block, error) {
	blkBytes, err := s.blockDB.Get(blkID[:])
	if err != nil {
		return nil, err
	}

	blkw := blkWrapper{}
	if _, err := Codec.Unmarshal(blkBytes, &blkw); err != nil {
		return nil, err
	}

	blk := timeBlock{}
	if _, err := Codec.Unmarshal(blkw.Blk, &blk); err != nil {
		return nil, err
	}

	blk.Initialize(blkw.Blk, blkw.Status, s.vm)

	s.blkCache.Put(blkID, blk)

	return &blk, nil
}

func (s *blockState) PutBlock(blk Block) error {
	blkw := blkWrapper{
		Blk:    blk.Bytes(),
		Status: blk.Status(),
		block:  blk,
	}

	bytes, err := Codec.Marshal(codecVersion, &blkw)
	if err != nil {
		return err
	}

	blkID := blk.ID()
	s.blkCache.Put(blkID, &blk)
	return s.blockDB.Put(blkID[:], bytes)
}

func (s *blockState) DeleteBlock(blkID ids.ID) error {
	s.blkCache.Put(blkID, nil)
	return s.blockDB.Delete(blkID[:])
}

func (s *blockState) GetLastAccepted() ids.ID             { return s.lastAccepted }
func (s *blockState) SetLastAccepted(lastAccepted ids.ID) { s.lastAccepted = lastAccepted }

func (s *blockState) ClearCache() {
	s.blkCache.Flush()
}
