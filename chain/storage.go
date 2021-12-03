package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/types"
)

// 0x0/ (singleton prefix info)
//   -> [reserved prefix]
// 0x1/ (prefix keys)
//   -> [reserved prefix]
//     -> [key]
// 0x2/ (tx hashes)
// 0x3/ (block hashes)

const (
	infoPrefix  = 0x0
	keyPrefix   = 0x1
	txPrefix    = 0x2
	blockPrefix = 0x3
)

var (
	lastAccepted = []byte("last_accepted")
)

func PrefixInfoKey(prefix []byte) []byte {
	return append([]byte{infoPrefix, types.PrefixDelimiter}, prefix...)
}

func PrefixValueKey(prefix []byte, key []byte) []byte {
	b := append([]byte{keyPrefix, types.PrefixDelimiter}, prefix...)
	b = append(b, types.PrefixDelimiter)
	return append(b, key...)
}

func PrefixTxKey(txID ids.ID) []byte {
	return append([]byte{txPrefix, types.PrefixDelimiter}, txID[:]...)
}

func PrefixBlockKey(blockID ids.ID) []byte {
	return append([]byte{blockPrefix, types.PrefixDelimiter}, blockID[:]...)
}

func GetPrefixInfo(db database.Database, prefix []byte) (*types.PrefixInfo, bool, error) {
	k := PrefixInfoKey(prefix)
	has, err := db.Has(k)
	if err != nil {
		return nil, false, err
	}
	if !has {
		return nil, false, nil
	}
	v, err := db.Get(k)
	if err != nil {
		return nil, false, err
	}
	var i types.PrefixInfo
	if _, err := codec.Unmarshal(v, &i); err != nil {
		return nil, false, err
	}
	return &i, true, nil
}

func GetValue(db database.Database, prefix []byte, key []byte) ([]byte, bool, error) {
	k := PrefixValueKey(prefix, key)
	has, err := db.Has(k)
	if err != nil {
		return nil, false, err
	}
	if !has {
		return nil, false, nil
	}
	v, err := db.Get(k)
	if err != nil {
		return nil, false, err
	}
	return v, true, nil
}

func SetLastAccepted(db database.Database, block *Block) error {
	bid := block.ID()
	if err := db.Put(lastAccepted, bid[:]); err != nil {
		return err
	}
	return db.Put(PrefixBlockKey(bid), block.Bytes())
}

func GetLastAccepted(db database.Database) (ids.ID, error) {
	v, err := db.Get(lastAccepted)
	if err != nil {
		return ids.ID{}, err
	}
	return ids.ToID(v)
}

func GetBlock(db database.Database, bid ids.ID) (*Block, error) {
	v, err := db.Get(PrefixBlockKey(bid))
	if err != nil {
		return nil, err
	}
	var b Block
	if _, err := codec.Unmarshal(v, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// DB
func HasPrefix(database.Database, []byte) (bool, error)                { return false, nil }
func HasPrefixKey(database.Database, []byte, []byte) (bool, error)     { return false, nil }
func GetPrefixKey(database.Database, []byte, []byte) ([]byte, error)   { return nil, nil }
func PutPrefixInfo(database.Database, []byte, *types.PrefixInfo) error { return nil }
func PutPrefixKey(database.Database, []byte, []byte, []byte) error     { return nil }
func DeletePrefixKey(database.Database, []byte, []byte) error          { return nil }
func DeleteAllPrefixKeys(database.Database, []byte) error              { return nil }
func SetTransaction(database.Database, *Transaction) error             { return nil }
func GetTransaction() (*Transaction, error)                            { return nil, nil }
func PutBlock(*Block) error                                            { return nil }
