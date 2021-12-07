package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
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

	PrefixDelimiter = '/'
)

var (
	lastAccepted = []byte("last_accepted")
)

func PrefixInfoKey(prefix []byte) []byte {
	return append([]byte{infoPrefix, PrefixDelimiter}, prefix...)
}

func PrefixValueKey(prefix []byte, key []byte) []byte {
	b := append([]byte{keyPrefix, PrefixDelimiter}, prefix...)
	b = append(b, PrefixDelimiter)
	return append(b, key...)
}

func PrefixTxKey(txID ids.ID) []byte {
	return append([]byte{txPrefix, PrefixDelimiter}, txID[:]...)
}

func PrefixBlockKey(blockID ids.ID) []byte {
	return append([]byte{blockPrefix, PrefixDelimiter}, blockID[:]...)
}

func GetPrefixInfo(db database.Database, prefix []byte) (*PrefixInfo, bool, error) {
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
	var i PrefixInfo
	if _, err := Unmarshal(v, &i); err != nil {
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
	has, err := db.Has(lastAccepted)
	if err != nil {
		return ids.ID{}, err
	}
	if !has {
		return ids.ID{}, nil
	}
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
	if _, err := Unmarshal(v, &b); err != nil {
		return nil, err
	}
	b.st = choices.Accepted // if block on disk, it must've been accepted
	return &b, nil
}

// DB
func HasPrefix(db database.Database, prefix []byte) (bool, error) {
	k := PrefixInfoKey(prefix)
	return db.Has(k)
}
func HasPrefixKey(db database.Database, prefix []byte, key []byte) (bool, error) {
	k := PrefixValueKey(prefix, key)
	return db.Has(k)
}
func PutPrefixInfo(db database.Database, prefix []byte, i *PrefixInfo) error {
	k := PrefixInfoKey(prefix)
	b, err := Marshal(i)
	if err != nil {
		return err
	}
	return db.Put(k, b)
}
func PutPrefixKey(db database.Database, prefix []byte, key []byte, value []byte) error {
	k := PrefixValueKey(prefix, key)
	return db.Put(k, value)
}
func DeletePrefixKey(db database.Database, prefix []byte, key []byte) error {
	k := PrefixValueKey(prefix, key)
	return db.Delete(k)
}
func DeleteAllPrefixKeys(db database.Database, prefix []byte) error {
	return database.ClearPrefix(db, db, PrefixValueKey(prefix, nil))
}
func SetTransaction(db database.Database, tx *Transaction) error {
	k := PrefixTxKey(tx.ID())
	b, err := Marshal(tx)
	if err != nil {
		return err
	}
	return db.Put(k, b)
}
func HasTransaction(db database.Database, txID ids.ID) (bool, error) {
	k := PrefixTxKey(txID)
	return db.Has(k)
}
