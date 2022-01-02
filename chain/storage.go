// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"errors"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/quarkvm/parser"
)

// 0x0/ (prefix mapping)
//   -> [user prefix] -> [raw prefix]
// 0x1/ (singleton prefix info)
//   -> [raw prefix]
// 0x2/ (prefix keys)
//   -> [raw prefix]
//     -> [key]
// 0x3/ (tx hashes)
// 0x4/ (block hashes)
// 0x5/ (prefix expiry queue)
//   -> [raw prefix]
// 0x6/ (prefix pruning queue)
//   -> [raw prefix]

const (
	mappingPrefix = 0x0
	infoPrefix    = 0x1
	keyPrefix     = 0x2
	txPrefix      = 0x3
	blockPrefix   = 0x4
	// prefixExpiryQueue  = 0x5
	// prefixPruningQueue = 0x6
)

var lastAccepted = []byte("last_accepted")

// TODO: use indirection to automatically service prefix->rawPrefix translation
// TODO: derive rawPrefix deterministically by hash(block hash + prefix)
func PrefixMappingKey(prefix []byte) (k []byte) {
	k = make([]byte, 2+len(prefix))
	k[0] = mappingPrefix
	k[1] = parser.Delimiter
	copy(k[2:], prefix)
	return k
}

// TODO: make ids.ID?
func PrefixInfoKey(rawPrefix []byte) (k []byte) {
	k = make([]byte, 2+len(rawPrefix))
	k[0] = infoPrefix
	k[1] = parser.Delimiter
	copy(k[2:], rawPrefix)
	return k
}

func PrefixValueKey(rawPrefix []byte, key []byte) (k []byte) {
	prefixN, keyN := len(rawPrefix), len(key)
	// TODO: can we not introduce an invariant that the delimiter is never
	// included?
	pfxDelimExists := bytes.HasSuffix(rawPrefix, []byte{parser.Delimiter})

	n := 2 + prefixN + keyN
	if !pfxDelimExists {
		n++
	}

	k = make([]byte, n)
	k[0] = keyPrefix
	k[1] = parser.Delimiter
	cur := 2

	copy(k[cur:], rawPrefix)
	cur += prefixN

	if !pfxDelimExists {
		k[cur] = parser.Delimiter
		cur++
	}
	if len(key) == 0 {
		return k
	}

	copy(k[cur:], key)
	return k
}

func PrefixTxKey(txID ids.ID) (k []byte) {
	k = make([]byte, 2+len(txID))
	k[0] = txPrefix
	k[1] = parser.Delimiter
	copy(k[2:], txID[:])
	return k
}

func PrefixBlockKey(blockID ids.ID) (k []byte) {
	k = make([]byte, 2+len(blockID))
	k[0] = blockPrefix
	k[1] = parser.Delimiter
	copy(k[2:], blockID[:])
	return k
}

func GetPrefixMapping(db database.KeyValueReader, prefix []byte) ([]byte, bool, error) {
	k := PrefixMappingKey(prefix)
	v, err := db.Get(k)
	if errors.Is(err, database.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return v, true, err
}

func GetPrefixInfo(db database.KeyValueReader, prefix []byte) (*PrefixInfo, bool, error) {
	rawPrefix, exists, err := GetPrefixMapping(db, prefix)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}

	k := PrefixInfoKey(rawPrefix)
	v, err := db.Get(k)
	if errors.Is(err, database.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var i PrefixInfo
	_, err = Unmarshal(v, &i)
	return &i, true, err
}

func GetValue(db database.KeyValueReader, prefix []byte, key []byte) ([]byte, bool, error) {
	rawPrefix, exists, err := GetPrefixMapping(db, prefix)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}

	k := PrefixValueKey(rawPrefix, key)
	v, err := db.Get(k)
	if errors.Is(err, database.ErrNotFound) {
		return nil, false, nil
	}
	return v, true, err
}

func SetLastAccepted(db database.KeyValueWriter, block *StatelessBlock) error {
	bid := block.ID()
	if err := db.Put(lastAccepted, bid[:]); err != nil {
		return err
	}
	return db.Put(PrefixBlockKey(bid), block.Bytes())
}

func HasLastAccepted(db database.Database) (bool, error) {
	return db.Has(lastAccepted)
}

func GetLastAccepted(db database.KeyValueReader) (ids.ID, error) {
	v, err := db.Get(lastAccepted)
	if errors.Is(err, database.ErrNotFound) {
		return ids.ID{}, nil
	}
	if err != nil {
		return ids.ID{}, err
	}
	return ids.ToID(v)
}

func GetBlock(db database.KeyValueReader, bid ids.ID) ([]byte, error) {
	return db.Get(PrefixBlockKey(bid))
}

// DB
func HasPrefix(db database.KeyValueReader, prefix []byte) (bool, error) {
	k := PrefixMappingKey(prefix)
	return db.Has(k)
}

func HasPrefixKey(db database.KeyValueReader, prefix []byte, key []byte) (bool, error) {
	rawPrefix, exists, err := GetPrefixMapping(db, prefix)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	k := PrefixValueKey(rawPrefix, key)
	return db.Has(k)
}

func PutPrefixInfo(db database.KeyValueWriter, prefix []byte, i *PrefixInfo) error {
	// TODO: handle need to now read on writes
	rawPrefix, exists, err := GetPrefixMapping(db, prefix)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("TODO")
	}

	k := PrefixInfoKey(rawPrefix)
	b, err := Marshal(i)
	if err != nil {
		return err
	}
	return db.Put(k, b)
}

func PutPrefixKey(db database.KeyValueWriter, prefix []byte, key []byte, value []byte) error {
	k := PrefixValueKey(prefix, key)
	return db.Put(k, value)
}

func DeletePrefixKey(db database.KeyValueWriter, prefix []byte, key []byte) error {
	k := PrefixValueKey(prefix, key)
	return db.Delete(k)
}

func DeleteAllPrefixKeys(db database.Database, prefix []byte) error {
	return database.ClearPrefix(db, db, PrefixValueKey(prefix, nil))
}

func SetTransaction(db database.KeyValueWriter, tx *Transaction) error {
	k := PrefixTxKey(tx.ID())
	b, err := Marshal(tx)
	if err != nil {
		return err
	}
	return db.Put(k, b)
}

func HasTransaction(db database.KeyValueReader, txID ids.ID) (bool, error) {
	k := PrefixTxKey(txID)
	return db.Has(k)
}

type KeyValue struct {
	Key   []byte `serialize:"true" json:"key"`
	Value []byte `serialize:"true" json:"value"`
}

// Range reads keys from the store.
// TODO: check prefix info to restrict reads to the owner?
func Range(db database.Database, prefix []byte, key []byte, opts ...OpOption) (kvs []KeyValue) {
	ret := &Op{key: PrefixValueKey(prefix, key)}
	ret.applyOpts(opts)

	startKey := ret.key
	var endKey []byte
	if len(ret.rangeEnd) > 0 {
		// set via "WithPrefix"
		endKey = ret.rangeEnd
		if !bytes.HasPrefix(endKey, []byte{keyPrefix, parser.Delimiter}) {
			// if overwritten via "WithRange"
			endKey = PrefixValueKey(prefix, endKey)
		}
	}

	kvs = make([]KeyValue, 0)
	cursor := db.NewIteratorWithStart(startKey)
	for cursor.Next() {
		if ret.rangeLimit > 0 && len(kvs) == int(ret.rangeLimit) {
			break
		}

		curKey := cursor.Key()

		comp := bytes.Compare(startKey, curKey)
		if comp == 0 { // startKey == curKey
			kvs = append(kvs, KeyValue{
				Key: bytes.Replace(
					curKey,
					[]byte{keyPrefix, parser.Delimiter},
					nil,
					1,
				),
				Value: cursor.Value(),
			})
			continue
		}
		if comp < -1 { // startKey < curKey; continue search
			continue
		}

		// startKey > curKey; continue search iff no range end is specified
		if len(endKey) == 0 {
			break
		}
		if bytes.Compare(curKey, endKey) >= 0 { // curKey > endKey
			break
		}

		kvs = append(kvs, KeyValue{
			Key:   bytes.Replace(curKey, []byte{keyPrefix, parser.Delimiter}, nil, 1),
			Value: cursor.Value(),
		})
	}
	return kvs
}

type Op struct {
	key        []byte
	rangeEnd   []byte
	rangeLimit uint32
}

type OpOption func(*Op)

func (op *Op) applyOpts(opts []OpOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func WithPrefix() OpOption {
	return func(op *Op) {
		op.rangeEnd = parser.GetRangeEnd(op.key)
	}
}

// Queries range [start,end).
func WithRangeEnd(end []byte) OpOption {
	return func(op *Op) { op.rangeEnd = end }
}

func WithRangeLimit(limit uint32) OpOption {
	return func(op *Op) { op.rangeLimit = limit }
}
