// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/quarkvm/parser"
)

// TODO: cleanup mapping diagram
// 0x0/ (block hashes)
// 0x1/ (tx hashes)
// 0x2/ (tx values)
// 0x3/ (singleton prefix info)
//   -> [prefix]:[prefix info/raw prefix]
// 0x4/ (prefix keys)
//   -> [raw prefix]
//     -> [key]
// 0x5/ (prefix expiry queue)
//   -> [raw prefix]
// 0x6/ (prefix pruning queue)
//   -> [raw prefix]

const (
	blockPrefix   = 0x0
	txPrefix      = 0x1
	txValuePrefix = 0x2
	infoPrefix    = 0x3
	keyPrefix     = 0x4
	expiryPrefix  = 0x5
	pruningPrefix = 0x6

	shortIDLen = 20
)

var lastAccepted = []byte("last_accepted")

// [blockPrefix] + [delimiter] + [blockID]
func PrefixBlockKey(blockID ids.ID) (k []byte) {
	k = make([]byte, 2+len(blockID))
	k[0] = blockPrefix
	k[1] = parser.Delimiter
	copy(k[2:], blockID[:])
	return k
}

// [txPrefix] + [delimiter] + [txID]
func PrefixTxKey(txID ids.ID) (k []byte) {
	k = make([]byte, 2+len(txID))
	k[0] = txPrefix
	k[1] = parser.Delimiter
	copy(k[2:], txID[:])
	return k
}

// [txValuePrefix] + [delimiter] + [txID]
func PrefixTxValueKey(txID ids.ID) (k []byte) {
	k = make([]byte, 2+len(txID))
	k[0] = txValuePrefix
	k[1] = parser.Delimiter
	copy(k[2:], txID[:])
	return k
}

// [infoPrefix] + [delimiter] + [prefix]
func PrefixInfoKey(prefix []byte) (k []byte) {
	k = make([]byte, 2+len(prefix))
	k[0] = infoPrefix
	k[1] = parser.Delimiter
	copy(k[2:], prefix)
	return k
}

func RawPrefix(prefix []byte, blockTime uint64) (ids.ShortID, error) {
	prefixLen := len(prefix)
	r := make([]byte, prefixLen+1+8)
	copy(r, prefix)
	r[prefixLen] = parser.Delimiter
	binary.BigEndian.PutUint64(r[prefixLen+1:], blockTime)
	h := hashing.ComputeHash160(r)
	rprefix, err := ids.ToShortID(h)
	if err != nil {
		return ids.ShortID{}, err
	}
	return rprefix, nil
}

// Assumes [prefix] and [key] do not contain delimiter
// [keyPrefix] + [delimiter] + [rawPrefix] + [delimiter] + [key]
func PrefixValueKey(rprefix ids.ShortID, key []byte) (k []byte) {
	k = make([]byte, 2+shortIDLen+1+len(key))
	k[0] = keyPrefix
	k[1] = parser.Delimiter
	copy(k[2:], rprefix[:])
	k[2+shortIDLen] = parser.Delimiter
	copy(k[2+shortIDLen+1:], key)
	return k
}

// [expiry/pruningPrefix] + [delimiter] + [timestamp] + [delimiter]
func RangeTimeKey(p byte, t uint64) (k []byte) {
	k = make([]byte, 2+8+1)
	k[0] = p
	k[1] = parser.Delimiter
	binary.BigEndian.PutUint64(k[2:], t)
	k[2+8] = parser.Delimiter
	return k
}

// [expiryPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawPrefix]
func PrefixExpiryKey(expiry uint64, rprefix ids.ShortID) (k []byte) {
	return specificTimeKey(expiryPrefix, expiry, rprefix)
}

// [pruningPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawPrefix]
func PrefixPruningKey(expired uint64, rprefix ids.ShortID) (k []byte) {
	return specificTimeKey(pruningPrefix, expired, rprefix)
}

const specificTimeKeyLen = 2 + 8 + 1 + shortIDLen

// [expiry/pruningPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawPrefix]
func specificTimeKey(p byte, t uint64, rprefix ids.ShortID) (k []byte) {
	k = make([]byte, specificTimeKeyLen)
	k[0] = p
	k[1] = parser.Delimiter
	binary.BigEndian.PutUint64(k[2:], t)
	k[2+8] = parser.Delimiter
	copy(k[2+8+1:], rprefix[:])
	return k
}

var ErrInvalidKeyFormat = errors.New("invalid key format")

// extracts expiry/pruning timstamp and raw prefix
func extractSpecificTimeKey(k []byte) (timestamp uint64, rprefix ids.ShortID, err error) {
	if len(k) != specificTimeKeyLen {
		return 0, ids.ShortEmpty, ErrInvalidKeyFormat
	}
	timestamp = binary.BigEndian.Uint64(k[2 : 2+8])
	rprefix, err = ids.ToShortID(k[2+8+1:])
	return timestamp, rprefix, err
}

func GetPrefixInfo(db database.KeyValueReader, prefix []byte) (*PrefixInfo, bool, error) {
	// [infoPrefix] + [delimiter] + [prefix]
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

	// [keyPrefix] + [delimiter] + [rawPrefix] + [delimiter] + [key]
	k := PrefixValueKey(prefixInfo.RawPrefix, key)
	txid, err := db.Get(k)
	if errors.Is(err, database.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	// Lookup stored value
	txID, err := ids.ToID(txid)
	if err != nil {
		return nil, false, err
	}
	vk := PrefixTxValueKey(txID)
	v, err := db.Get(vk)
	if err != nil {
		return nil, false, err
	}
	return v, true, err
}

func extractAndStoreValues(db database.KeyValueWriter, block *StatelessBlock) ([]*Transaction, error) {
	oldTxs := make([]*Transaction, len(block.Txs))
	for i, tx := range block.Txs {
		switch t := tx.UnsignedTransaction.(type) {
		case *SetTx:
			if len(t.Value) == 0 {
				oldTxs[i] = tx
				continue
			}

			// Copy transaction for later
			cptx := tx.Copy()
			if err := cptx.Init(); err != nil {
				return nil, err
			}
			oldTxs[i] = cptx

			if err := db.Put(PrefixTxValueKey(tx.ID()), t.Value); err != nil {
				return nil, err
			}
			backup := make([]byte, len(t.Value))
			copy(backup, t.Value)
			t.Value = tx.id[:] // used to properly parse on restore
		default:
			oldTxs[i] = tx
		}
	}
	return oldTxs, nil
}

func restoreValues(db database.KeyValueReader, block *StatefulBlock) error {
	for _, tx := range block.Txs {
		switch t := tx.UnsignedTransaction.(type) {
		case *SetTx:
			if len(t.Value) == 0 {
				continue
			}
			txID, err := ids.ToID(t.Value)
			if err != nil {
				return err
			}
			b, err := db.Get(PrefixTxValueKey(txID))
			if err != nil {
				return err
			}
			t.Value = b
		}
	}
	return nil
}

func SetLastAccepted(db database.KeyValueWriter, block *StatelessBlock) error {
	bid := block.ID()
	if err := db.Put(lastAccepted, bid[:]); err != nil {
		return err
	}
	oldTxs, err := extractAndStoreValues(db, block)
	if err != nil {
		return err
	}
	nbytes, err := Marshal(block)
	if err != nil {
		return err
	}
	if err := db.Put(PrefixBlockKey(bid), nbytes); err != nil {
		return err
	}
	block.Txs = oldTxs
	return nil
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

func GetBlock(db database.KeyValueReader, bid ids.ID) (*StatefulBlock, []byte, error) {
	b, err := db.Get(PrefixBlockKey(bid))
	if err != nil {
		return nil, nil, err
	}
	blk := new(StatefulBlock)
	if _, err := Unmarshal(b, blk); err != nil {
		return nil, nil, err
	}
	if err := restoreValues(db, blk); err != nil {
		return nil, nil, err
	}
	fb, err := Marshal(blk)
	if err != nil {
		return nil, nil, err
	}
	return blk, fb, nil
}

// ExpireNext queries "expiryPrefix" key space to find expiring keys,
// deletes their prefixInfos, and schedules its key pruning with its raw prefix.
func ExpireNext(db database.Database, rparent int64, rcurrent int64, bootstrapped bool) (err error) {
	parent, current := uint64(rparent), uint64(rcurrent)
	startKey := RangeTimeKey(expiryPrefix, parent)
	endKey := RangeTimeKey(expiryPrefix, current)
	cursor := db.NewIteratorWithStart(startKey)
	for cursor.Next() {
		// [expiryPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawPrefix]
		curKey := cursor.Key()
		if bytes.Compare(startKey, curKey) < -1 { // startKey < curKey; continue search
			continue
		}
		if bytes.Compare(curKey, endKey) > 0 { // curKey > endKey; end search
			break
		}
		if err := db.Delete(cursor.Key()); err != nil {
			return err
		}

		// [prefix]
		pfx := cursor.Value()

		// [infoPrefix] + [delimiter] + [prefix]
		k := PrefixInfoKey(pfx)
		if err := db.Delete(k); err != nil {
			return err
		}
		expired, rpfx, err := extractSpecificTimeKey(curKey)
		if err != nil {
			return err
		}

		if bootstrapped {
			// [pruningPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawPrefix]
			k = PrefixPruningKey(expired, rpfx)
			if err := db.Put(k, nil); err != nil {
				return err
			}
		} else {
			// If we are not yet bootstrapped, we should delete the dangling value keys
			// immediately instead of clearing async.
			if err := database.ClearPrefix(db, db, PrefixValueKey(rpfx, nil)); err != nil {
				return err
			}
		}
		log.Debug("prefix expired", "prefix", string(pfx))
	}
	return nil
}

// PruneNext queries the keys that are currently marked with "pruningPrefix",
// and clears them from the database.
func PruneNext(db database.Database, limit int) (removals int, err error) {
	startKey := RangeTimeKey(pruningPrefix, 0)
	endKey := RangeTimeKey(pruningPrefix, math.MaxInt64)
	cursor := db.NewIteratorWithStart(startKey)
	for cursor.Next() && removals < limit {
		// [pruningPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawPrefix]
		curKey := cursor.Key()
		if bytes.Compare(startKey, curKey) < -1 { // startKey < curKey; continue search
			continue
		}
		if bytes.Compare(curKey, endKey) > 0 { // curKey > endKey; end search
			break
		}
		_, rpfx, err := extractSpecificTimeKey(curKey)
		if err != nil {
			return removals, err
		}
		if err := db.Delete(curKey); err != nil {
			return removals, err
		}
		// [keyPrefix] + [delimiter] + [rawPrefix] + [delimiter] + [key]
		if err := database.ClearPrefix(db, db, PrefixValueKey(rpfx, nil)); err != nil {
			return removals, err
		}
		log.Debug("rprefix pruned", "rprefix", rpfx.Hex())
		removals++
	}
	return removals, nil
}

// DB
func HasPrefix(db database.KeyValueReader, prefix []byte) (bool, error) {
	// [infoPrefix] + [delimiter] + [prefix]
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

	// [keyPrefix] + [delimiter] + [rawPrefix] + [delimiter] + [key]
	k := PrefixValueKey(prefixInfo.RawPrefix, key)
	return db.Has(k)
}

func PutPrefixInfo(db database.KeyValueWriter, prefix []byte, i *PrefixInfo, lastExpiry uint64) error {
	if i.RawPrefix == ids.ShortEmpty {
		rprefix, err := RawPrefix(prefix, i.Created)
		if err != nil {
			return err
		}
		i.RawPrefix = rprefix
	}
	if lastExpiry > 0 {
		// [expiryPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawPrefix]
		k := PrefixExpiryKey(lastExpiry, i.RawPrefix)
		if err := db.Delete(k); err != nil {
			return err
		}
	}
	// [expiryPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawPrefix]
	k := PrefixExpiryKey(i.Expiry, i.RawPrefix)
	if err := db.Put(k, prefix); err != nil {
		return err
	}
	// [infoPrefix] + [delimiter] + [prefix]
	k = PrefixInfoKey(prefix)
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
		return ErrPrefixMissing
	}
	// [keyPrefix] + [delimiter] + [rawPrefix] + [delimiter] + [key]
	k := PrefixValueKey(prefixInfo.RawPrefix, key)
	return db.Put(k, value)
}

func DeletePrefixKey(db database.Database, prefix []byte, key []byte) error {
	prefixInfo, exists, err := GetPrefixInfo(db, prefix)
	if err != nil {
		return err
	}
	if !exists {
		return ErrPrefixMissing
	}
	k := PrefixValueKey(prefixInfo.RawPrefix, key)
	return db.Delete(k)
}

func SetTransaction(db database.KeyValueWriter, tx *Transaction) error {
	k := PrefixTxKey(tx.ID())
	return db.Put(k, nil)
}

func HasTransaction(db database.KeyValueReader, txID ids.ID) (bool, error) {
	k := PrefixTxKey(txID)
	return db.Has(k)
}

type KeyValue struct {
	Key   []byte `serialize:"true" json:"key"`
	Value []byte `serialize:"true" json:"value"`
}

func handleCursorValue(db database.KeyValueReader, b []byte) ([]byte, error) {
	txID, err := ids.ToID(b)
	if err != nil {
		return nil, err
	}
	vk := PrefixTxValueKey(txID)
	v, err := db.Get(vk)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// Range reads keys from the store.
func Range(db database.Database, prefix []byte, key []byte, opts ...OpOption) (kvs []KeyValue, err error) {
	prefixInfo, exists, err := GetPrefixInfo(db, prefix)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrPrefixMissing
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

		// [keyPrefix] + [delimiter] + [rawPrefix] + [delimiter] + [key]
		curKey := cursor.Key()
		formattedKey := curKey[2+shortIDLen+1:]

		comp := bytes.Compare(startKey, curKey)
		if comp == 0 { // startKey == curKey
			v, err := handleCursorValue(db, cursor.Value())
			if err != nil {
				return nil, err
			}
			kvs = append(kvs, KeyValue{
				Key:   formattedKey,
				Value: v,
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

		v, err := handleCursorValue(db, cursor.Value())
		if err != nil {
			return nil, err
		}
		kvs = append(kvs, KeyValue{
			Key:   formattedKey,
			Value: v,
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
