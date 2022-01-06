// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/quarkvm/parser"
)

// TODO: cleanup mapping diagram
// 0x0/ (singleton prefix info)
//   -> [prefix]:[prefix info/raw prefix]
// 0x1/ (prefix keys)
//   -> [raw prefix]
//     -> [key]
// 0x2/ (tx hashes)
// 0x3/ (block hashes)
// 0x4/ (prefix expiry queue)
//   -> [raw prefix]
// 0x5/ (prefix pruning queue)
//   -> [raw prefix]

type rawPrefix ids.ShortID

const (
	infoPrefix  = 0x0
	keyPrefix   = 0x1
	txPrefix    = 0x2
	blockPrefix = 0x3

	// TODO: implement queues
	// prefixExpiryQueue  = 0x4
	// prefixPruningQueue = 0x5
)

var lastAccepted = []byte("last_accepted")

// TODO: move to right spot
var prefixMissing = errors.New("prefix missing")

func PrefixInfoKey(prefix []byte) (k []byte) {
	k = make([]byte, 2+len(prefix))
	k[0] = infoPrefix
	k[1] = parser.Delimiter
	copy(k[2:], prefix)
	return k
}

// Assumes [prefix] and [key] do not contain delimiter
func PrefixValueKey(prefix rawPrefix, key []byte) (k []byte) {
	k = make([]byte, 2+len(prefix)+2+len(key))
	k[0] = keyPrefix
	k[1] = parser.Delimiter
	copy(k[2:], prefix[:])
	k[2+len(prefix)] = parser.Delimiter
	copy(k[2+len(prefix)+1:], key)
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

func RawPrefix(prefix []byte, blockTime int64) (rawPrefix, error) {
	prefixLen := len(prefix)
	raw := make([]byte, prefixLen+1+binary.MaxVarintLen64)
	copy(raw, prefix)
	raw[prefixLen] = parser.Delimiter
	binary.PutVarint(raw[prefixLen+1:], blockTime)
	rp, err := ids.ToShortID(raw)
	if err != nil {
		// TODO: clean up casting
		return rawPrefix(ids.ShortID{}), err
	}
	return rawPrefix(rp), nil
}

func GetPrefixInfo(db database.KeyValueReader, prefix []byte) (*PrefixInfo, bool, error) {
	// TODO: add caching (will need some expiry when keys cleared)
	k := PrefixInfoKey(prefix)
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
	prefixInfo, exists, err := GetPrefixInfo(db, prefix)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}

	k := PrefixValueKey(prefixInfo.RawPrefix, key)
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
	k := PrefixInfoKey(prefix)
	return db.Has(k)
}

func HasPrefixKey(db database.KeyValueReader, prefix []byte, key []byte) (bool, error) {
	prefixInfo, exists, err := GetPrefixInfo(db, prefix)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	k := PrefixValueKey(prefixInfo.RawPrefix, key)
	return db.Has(k)
}

func PutPrefixInfo(db database.KeyValueWriter, prefix []byte, i *PrefixInfo) error {
	k := PrefixInfoKey(prefix)
	b, err := Marshal(i)
	if err != nil {
		return err
	}
	return db.Put(k, b)
}

func PutPrefixKey(db database.Database, prefix []byte, key []byte, value []byte) error {
	prefixInfo, exists, err := GetPrefixInfo(db, prefix)
	if err != nil {
		return err
	}
	if !exists {
		return prefixMissing
	}
	k := PrefixValueKey(prefixInfo.RawPrefix, key)
	return db.Put(k, value)
}

func DeletePrefixKey(db database.Database, prefix []byte, key []byte) error {
	prefixInfo, exists, err := GetPrefixInfo(db, prefix)
	if err != nil {
		return err
	}
	if !exists {
		return prefixMissing
	}
	k := PrefixValueKey(prefixInfo.RawPrefix, key)
	return db.Delete(k)
}

func DeleteAllPrefixKeys(db database.Database, prefix rawPrefix) error {
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
func Range(db database.Database, prefix []byte, key []byte, opts ...OpOption) (kvs []KeyValue, err error) {
	prefixInfo, exists, err := GetPrefixInfo(db, prefix)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, prefixMissing
	}
	ret := &Op{key: PrefixValueKey(prefixInfo.RawPrefix, key)}
	ret.applyOpts(opts)

	startKey := ret.key
	var endKey []byte
	if len(ret.rangeEnd) > 0 {
		// set via "WithPrefix"
		endKey = ret.rangeEnd
		if !bytes.HasPrefix(endKey, []byte{keyPrefix, parser.Delimiter}) {
			// if overwritten via "WithRange"
			endKey = PrefixValueKey(prefixInfo.RawPrefix, endKey)
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
	return kvs, nil
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
