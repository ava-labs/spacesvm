package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/wrappers"
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
	b := make([]byte, len(prefix)+2)
	packer := wrappers.Packer{Bytes: b}
	packer.PackByte(infoPrefix)
	packer.PackByte(PrefixDelimiter)
	packer.PackBytes(prefix)
	return b
}

func PrefixValueKey(prefix []byte, key []byte) []byte {
	b := make([]byte, len(prefix)+len(key)+3)
	packer := wrappers.Packer{Bytes: b}
	packer.PackByte(keyPrefix)
	packer.PackByte(PrefixDelimiter)
	packer.PackBytes(prefix)
	packer.PackByte(PrefixDelimiter)
	packer.PackBytes(key)
	return b
}

func PrefixTxKey(txID ids.ID) []byte {
	b := make([]byte, len(txID)+2)
	packer := wrappers.Packer{Bytes: b}
	packer.PackByte(txPrefix)
	packer.PackByte(PrefixDelimiter)
	packer.PackBytes(txID[:])
	return b
}

func PrefixBlockKey(blockID ids.ID) []byte {
	b := make([]byte, len(blockID)+2)
	packer := wrappers.Packer{Bytes: b}
	packer.PackByte(blockPrefix)
	packer.PackByte(PrefixDelimiter)
	packer.PackBytes(blockID[:])
	return b
}

func GetPrefixInfo(db database.Database, prefix []byte) (*PrefixInfo, bool, error) {
	k := PrefixInfoKey(prefix)
	v, err := db.Get(k)
	if err == database.ErrNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var i PrefixInfo
	_, err = Unmarshal(v, &i)
	return &i, true, err
}

func GetValue(db database.Database, prefix []byte, key []byte) ([]byte, bool, error) {
	k := PrefixValueKey(prefix, key)
	v, err := db.Get(k)
	if err == database.ErrNotFound {
		return nil, false, nil
	}
	return v, true, err
}

func SetLastAccepted(db database.Database, block *Block) error {
	bid := block.ID()
	if err := db.Put(lastAccepted, bid[:]); err != nil {
		return err
	}
	return db.Put(PrefixBlockKey(bid), block.Bytes())
}

func HasLastAccepted(db database.Database) (bool, error) {
	return db.Has(lastAccepted)
}

func GetLastAccepted(db database.Database) (ids.ID, error) {
	v, err := db.Get(lastAccepted)
	if err != nil {
		return ids.ID{}, err
	}
	return ids.ToID(v)
}

func GetBlock(db database.Database, bid ids.ID) ([]byte, error) {
	return db.Get(PrefixBlockKey(bid))
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
