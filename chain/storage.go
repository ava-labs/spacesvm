// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/ava-labs/avalanchego/cache"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ethereum/go-ethereum/common"
	smath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/spacesvm/parser"
)

// 0x0/ (block hashes)
// 0x1/ (tx hashes)
//   -> [tx hash]=>nil
// 0x2/ (tx values)
//   -> [tx hash]=>value
// 0x3/ (singleton space info)
//   -> [space]:[space info/raw space]
// 0x4/ (space keys)
//   -> [raw space]
//     -> [key]
// 0x5/ (space expiry queue)
//   -> [raw space]
// 0x6/ (space pruning queue)
//   -> [raw space]
// 0x7/ (balance)
//   -> [owner]=> balance
// 0x8/ (owned spaces)
//   -> [owner]/[space]=> nil

const (
	blockPrefix   = 0x0
	txPrefix      = 0x1
	txValuePrefix = 0x2
	infoPrefix    = 0x3
	keyPrefix     = 0x4
	expiryPrefix  = 0x5
	pruningPrefix = 0x6
	balancePrefix = 0x7
	ownedPrefix   = 0x8

	shortIDLen = 20

	linkedTxLRUSize = 512
)

type CompactRange struct {
	Start []byte
	Limit []byte
}

var (
	lastAccepted  = []byte("last_accepted")
	linkedTxCache = &cache.LRU{Size: linkedTxLRUSize}

	CompactRanges = []*CompactRange{
		// Don't compact block/tx/txValue ranges because no overwriting/deletion
		{[]byte{infoPrefix, parser.ByteDelimiter}, []byte{keyPrefix, parser.ByteDelimiter}},
		{[]byte{keyPrefix, parser.ByteDelimiter}, []byte{expiryPrefix, parser.ByteDelimiter}},
		// Group expiry and pruning together
		{[]byte{expiryPrefix, parser.ByteDelimiter}, []byte{balancePrefix, parser.ByteDelimiter}},
		{[]byte{balancePrefix, parser.ByteDelimiter}, []byte{ownedPrefix, parser.ByteDelimiter}},
		{[]byte{ownedPrefix, parser.ByteDelimiter}, []byte{ownedPrefix + 1, parser.ByteDelimiter}},
	}
)

// [blockPrefix] + [delimiter] + [blockID]
func PrefixBlockKey(blockID ids.ID) (k []byte) {
	k = make([]byte, 2+len(blockID))
	k[0] = blockPrefix
	k[1] = parser.ByteDelimiter
	copy(k[2:], blockID[:])
	return k
}

// [txPrefix] + [delimiter] + [txID]
func PrefixTxKey(txID ids.ID) (k []byte) {
	k = make([]byte, 2+len(txID))
	k[0] = txPrefix
	k[1] = parser.ByteDelimiter
	copy(k[2:], txID[:])
	return k
}

// [txValuePrefix] + [delimiter] + [txID]
func PrefixTxValueKey(txID ids.ID) (k []byte) {
	k = make([]byte, 2+len(txID))
	k[0] = txValuePrefix
	k[1] = parser.ByteDelimiter
	copy(k[2:], txID[:])
	return k
}

// [infoPrefix] + [delimiter] + [space]
func SpaceInfoKey(space []byte) (k []byte) {
	k = make([]byte, 2+len(space))
	k[0] = infoPrefix
	k[1] = parser.ByteDelimiter
	copy(k[2:], space)
	return k
}

func RawSpace(space []byte, blockTime uint64) (ids.ShortID, error) {
	spaceLen := len(space)
	r := make([]byte, spaceLen+1+8)
	copy(r, space)
	r[spaceLen] = parser.ByteDelimiter
	binary.BigEndian.PutUint64(r[spaceLen+1:], blockTime)
	h := hashing.ComputeHash160(r)
	rspace, err := ids.ToShortID(h)
	if err != nil {
		return ids.ShortID{}, err
	}
	return rspace, nil
}

// Assumes [space] and [key] do not contain delimiter
// [keyPrefix] + [delimiter] + [rawSpace] + [delimiter] + [key]
func SpaceValueKey(rspace ids.ShortID, key []byte) (k []byte) {
	k = make([]byte, 2+shortIDLen+1+len(key))
	k[0] = keyPrefix
	k[1] = parser.ByteDelimiter
	copy(k[2:], rspace[:])
	k[2+shortIDLen] = parser.ByteDelimiter
	copy(k[2+shortIDLen+1:], key)
	return k
}

// [expiry/pruningPrefix] + [delimiter] + [timestamp] + [delimiter]
func RangeTimeKey(p byte, t uint64) (k []byte) {
	k = make([]byte, 2+8+1)
	k[0] = p
	k[1] = parser.ByteDelimiter
	binary.BigEndian.PutUint64(k[2:], t)
	k[2+8] = parser.ByteDelimiter
	return k
}

// [expiryPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawSpace]
func PrefixExpiryKey(expiry uint64, rspace ids.ShortID) (k []byte) {
	return specificTimeKey(expiryPrefix, expiry, rspace)
}

// [pruningPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawSpace]
func PrefixPruningKey(expired uint64, rspace ids.ShortID) (k []byte) {
	return specificTimeKey(pruningPrefix, expired, rspace)
}

// [balancePrefix] + [delimiter] + [address]
func PrefixBalanceKey(address common.Address) (k []byte) {
	k = make([]byte, 2+common.AddressLength)
	k[0] = balancePrefix
	k[1] = parser.ByteDelimiter
	copy(k[2:], address[:])
	return
}

// [ownedPrefix] + [delimiter] + [address] + [delimiter] + [space]
func PrefixOwnedKey(address common.Address, space []byte) (k []byte) {
	k = make([]byte, 2+common.AddressLength+1+len(space))
	k[0] = ownedPrefix
	k[1] = parser.ByteDelimiter
	copy(k[2:], address[:])
	k[2+common.AddressLength] = parser.ByteDelimiter
	copy(k[2+common.AddressLength+1:], space)
	return
}

const specificTimeKeyLen = 2 + 8 + 1 + shortIDLen

// [expiry/pruningPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawSpace]
func specificTimeKey(p byte, t uint64, rspace ids.ShortID) (k []byte) {
	k = make([]byte, specificTimeKeyLen)
	k[0] = p
	k[1] = parser.ByteDelimiter
	binary.BigEndian.PutUint64(k[2:], t)
	k[2+8] = parser.ByteDelimiter
	copy(k[2+8+1:], rspace[:])
	return k
}

var ErrInvalidKeyFormat = errors.New("invalid key format")

// extracts expiry/pruning timstamp and raw space
func extractSpecificTimeKey(k []byte) (timestamp uint64, rspace ids.ShortID, err error) {
	if len(k) != specificTimeKeyLen {
		return 0, ids.ShortEmpty, ErrInvalidKeyFormat
	}
	timestamp = binary.BigEndian.Uint64(k[2 : 2+8])
	rspace, err = ids.ToShortID(k[2+8+1:])
	return timestamp, rspace, err
}

func GetSpaceInfo(db database.KeyValueReader, space []byte) (*SpaceInfo, bool, error) {
	// [infoPrefix] + [delimiter] + [space]
	k := SpaceInfoKey(space)
	v, err := db.Get(k)
	if errors.Is(err, database.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var i SpaceInfo
	_, err = Unmarshal(v, &i)
	return &i, true, err
}

func GetValueMeta(db database.KeyValueReader, space []byte, key []byte) (*ValueMeta, bool, error) {
	spaceInfo, exists, err := GetSpaceInfo(db, space)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}

	// [keyPrefix] + [delimiter] + [rawSpace] + [delimiter] + [key]
	k := SpaceValueKey(spaceInfo.RawSpace, key)
	rvmeta, err := db.Get(k)
	if errors.Is(err, database.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	vmeta := new(ValueMeta)
	if _, err := Unmarshal(rvmeta, vmeta); err != nil {
		return nil, false, err
	}
	return vmeta, true, nil
}

func GetValue(db database.KeyValueReader, space []byte, key []byte) ([]byte, bool, error) {
	spaceInfo, exists, err := GetSpaceInfo(db, space)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}

	// [keyPrefix] + [delimiter] + [rawSpace] + [delimiter] + [key]
	k := SpaceValueKey(spaceInfo.RawSpace, key)
	rvmeta, err := db.Get(k)
	if errors.Is(err, database.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	vmeta := new(ValueMeta)
	if _, err := Unmarshal(rvmeta, vmeta); err != nil {
		return nil, false, err
	}

	// Lookup stored value
	v, err := getLinkedValue(db, vmeta.TxID[:])
	if err != nil {
		return nil, false, err
	}
	return v, true, err
}

type KeyValueMeta struct {
	Key       string     `serialize:"true" json:"key"`
	ValueMeta *ValueMeta `serialize:"true" json:"valueMeta"`
}

func GetAllValueMetas(db database.Database, rspace ids.ShortID) (kvs []*KeyValueMeta, err error) {
	baseKey := SpaceValueKey(rspace, nil)
	cursor := db.NewIteratorWithStart(baseKey)
	defer cursor.Release()
	kvs = []*KeyValueMeta{}
	for cursor.Next() {
		curKey := cursor.Key()
		if bytes.Compare(baseKey, curKey) < -1 { // startKey < curKey; continue search
			continue
		}
		if !bytes.Contains(curKey, baseKey) { // curKey does not contain base key; end search
			break
		}

		vmeta := new(ValueMeta)
		if _, err := Unmarshal(cursor.Value(), vmeta); err != nil {
			return nil, err
		}

		kvs = append(kvs, &KeyValueMeta{
			// [keyPrefix] + [delimiter] + [rawSpace] + [delimiter] + [key]
			Key:       string(curKey[2+shortIDLen+1:]),
			ValueMeta: vmeta,
		})
	}
	return kvs, cursor.Error()
}

// linkValues extracts all *SetTx.Value in [block] and replaces them with the
// corresponding txID where they were found. The extracted value is then
// written to disk.
func linkValues(db database.KeyValueWriter, block *StatelessBlock) ([]*Transaction, error) {
	g := block.vm.Genesis()
	ogTxs := make([]*Transaction, len(block.Txs))
	for i, tx := range block.Txs {
		switch t := tx.UnsignedTransaction.(type) {
		case *SetTx:
			if len(t.Value) == 0 {
				ogTxs[i] = tx
				continue
			}

			// Copy transaction for later
			cptx := tx.Copy()
			if err := cptx.Init(g); err != nil {
				return nil, err
			}
			ogTxs[i] = cptx

			if err := db.Put(PrefixTxValueKey(tx.ID()), t.Value); err != nil {
				return nil, err
			}
			t.Value = tx.id[:] // used to properly parse on restore
		default:
			ogTxs[i] = tx
		}
	}
	return ogTxs, nil
}

// restoreValues restores the unlinked values associated with all *SetTx.Value
// in [block].
func restoreValues(db database.KeyValueReader, block *StatefulBlock) error {
	for _, tx := range block.Txs {
		if t, ok := tx.UnsignedTransaction.(*SetTx); ok {
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
	ogTxs, err := linkValues(db, block)
	if err != nil {
		return err
	}
	sbytes, err := Marshal(block.StatefulBlock)
	if err != nil {
		return err
	}
	if err := db.Put(PrefixBlockKey(bid), sbytes); err != nil {
		return err
	}
	// Restore the original transactions in the block in case it is cached for
	// later use.
	block.Txs = ogTxs
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

func GetBlock(db database.KeyValueReader, bid ids.ID) (*StatefulBlock, error) {
	b, err := db.Get(PrefixBlockKey(bid))
	if err != nil {
		return nil, err
	}
	blk := new(StatefulBlock)
	if _, err := Unmarshal(b, blk); err != nil {
		return nil, err
	}
	if err := restoreValues(db, blk); err != nil {
		return nil, err
	}
	return blk, nil
}

// ExpireNext queries "expiryPrefix" key space to find expiring keys,
// deletes their spaceInfos, and schedules its key pruning with its raw space.
func ExpireNext(db database.Database, rparent int64, rcurrent int64, bootstrapped bool) (err error) {
	parent, current := uint64(rparent), uint64(rcurrent)
	startKey := RangeTimeKey(expiryPrefix, parent)
	endKey := RangeTimeKey(expiryPrefix, current)
	cursor := db.NewIteratorWithStart(startKey)
	defer cursor.Release()
	for cursor.Next() {
		// [expiryPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawSpace]
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

		// [owner] + [space]
		expiryValue := cursor.Value()
		owner := common.BytesToAddress(expiryValue[:common.AddressLength])
		space := expiryValue[common.AddressLength:]

		// Update owned prefix
		if err := db.Delete(PrefixOwnedKey(owner, space)); err != nil {
			return err
		}

		// [infoPrefix] + [delimiter] + [space]
		k := SpaceInfoKey(space)
		if err := db.Delete(k); err != nil {
			return err
		}

		expired, rspc, err := extractSpecificTimeKey(curKey)
		if err != nil {
			return err
		}
		if bootstrapped {
			// [pruningPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawSpace]
			k = PrefixPruningKey(expired, rspc)
			if err := db.Put(k, nil); err != nil {
				return err
			}
		} else {
			// If we are not yet bootstrapped, we should delete the dangling value keys
			// immediately instead of clearing async.
			if err := database.ClearPrefix(db, db, SpaceValueKey(rspc, nil)); err != nil {
				return err
			}
		}
		log.Debug("space expired", "space", string(space))
	}
	return cursor.Error()
}

// PruneNext queries the keys that are currently marked with "pruningPrefix",
// and clears them from the database.
func PruneNext(db database.Database, limit int) (removals int, err error) {
	startKey := RangeTimeKey(pruningPrefix, 0)
	endKey := RangeTimeKey(pruningPrefix, math.MaxInt64)
	cursor := db.NewIteratorWithStart(startKey)
	defer cursor.Release()
	for cursor.Next() && removals < limit {
		// [pruningPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawSpace]
		curKey := cursor.Key()
		if bytes.Compare(startKey, curKey) < -1 { // startKey < curKey; continue search
			continue
		}
		if bytes.Compare(curKey, endKey) > 0 { // curKey > endKey; end search
			break
		}
		_, rspc, err := extractSpecificTimeKey(curKey)
		if err != nil {
			return removals, err
		}
		if err := db.Delete(curKey); err != nil {
			return removals, err
		}
		// [keyPrefix] + [delimiter] + [rawSpace] + [delimiter] + [key]
		if err := database.ClearPrefix(db, db, SpaceValueKey(rspc, nil)); err != nil {
			return removals, err
		}
		log.Debug("rspace pruned", "rspace", rspc.Hex())
		removals++
	}
	return removals, cursor.Error()
}

// DB
func HasSpace(db database.KeyValueReader, space []byte) (bool, error) {
	// [infoPrefix] + [delimiter] + [space]
	k := SpaceInfoKey(space)
	return db.Has(k)
}

func HasSpaceKey(db database.KeyValueReader, space []byte, key []byte) (bool, error) {
	spaceInfo, exists, err := GetSpaceInfo(db, space)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	// [keyPrefix] + [delimiter] + [rawSpace] + [delimiter] + [key]
	k := SpaceValueKey(spaceInfo.RawSpace, key)
	return db.Has(k)
}

func ExpiryDataValue(address common.Address, space []byte) (v []byte) {
	v = make([]byte, common.AddressLength+len(space))
	copy(v, address[:])
	copy(v[common.AddressLength:], space)
	return v
}

func PutSpaceInfo(db database.KeyValueWriterDeleter, space []byte, i *SpaceInfo, lastExpiry uint64) error {
	// If [RawSpace] is empty, this is a new space.
	if i.RawSpace == ids.ShortEmpty {
		rspace, err := RawSpace(space, i.Created)
		if err != nil {
			return err
		}
		i.RawSpace = rspace

		// Only store the owner on creation
		if err := db.Put(PrefixOwnedKey(i.Owner, space), nil); err != nil {
			return err
		}
	}
	if lastExpiry > 0 {
		// [expiryPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawSpace]
		k := PrefixExpiryKey(lastExpiry, i.RawSpace)
		if err := db.Delete(k); err != nil {
			return err
		}
	}
	// [expiryPrefix] + [delimiter] + [timestamp] + [delimiter] + [rawSpace]
	k := PrefixExpiryKey(i.Expiry, i.RawSpace)
	if err := db.Put(k, ExpiryDataValue(i.Owner, space)); err != nil {
		return err
	}
	// [infoPrefix] + [delimiter] + [space]
	k = SpaceInfoKey(space)
	b, err := Marshal(i)
	if err != nil {
		return err
	}
	return db.Put(k, b)
}

// MoveSpaceInfo should only be used if the expiry isn't changing and
// [SpaceInfo] is already in the database.
func MoveSpaceInfo(
	db database.KeyValueWriterDeleter, oldOwner common.Address,
	space []byte, i *SpaceInfo,
) error {
	// [infoPrefix] + [delimiter] + [space]
	k := SpaceInfoKey(space)
	b, err := Marshal(i)
	if err != nil {
		return err
	}
	if err := db.Put(k, b); err != nil {
		return err
	}
	// Updated owned prefix
	if err := db.Delete(PrefixOwnedKey(oldOwner, space)); err != nil {
		return err
	}
	if err := db.Put(PrefixOwnedKey(i.Owner, space), nil); err != nil {
		return err
	}
	k = PrefixExpiryKey(i.Expiry, i.RawSpace)
	return db.Put(k, ExpiryDataValue(i.Owner, space))
}

type ValueMeta struct {
	Size uint64 `serialize:"true" json:"size"`
	TxID ids.ID `serialize:"true" json:"txId"`

	Created uint64 `serialize:"true" json:"created"`
	Updated uint64 `serialize:"true" json:"updated"`
}

func PutSpaceKey(db database.KeyValueReaderWriter, space []byte, key []byte, vmeta *ValueMeta) error {
	spaceInfo, exists, err := GetSpaceInfo(db, space)
	if err != nil {
		return err
	}
	if !exists {
		return ErrSpaceMissing
	}
	// [keyPrefix] + [delimiter] + [rawSpace] + [delimiter] + [key]
	k := SpaceValueKey(spaceInfo.RawSpace, key)
	rvmeta, err := Marshal(vmeta)
	if err != nil {
		return err
	}
	return db.Put(k, rvmeta)
}

func DeleteSpaceKey(db database.Database, space []byte, key []byte) error {
	spaceInfo, exists, err := GetSpaceInfo(db, space)
	if err != nil {
		return err
	}
	if !exists {
		return ErrSpaceMissing
	}
	k := SpaceValueKey(spaceInfo.RawSpace, key)
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

func getLinkedValue(db database.KeyValueReader, b []byte) ([]byte, error) {
	bh := string(b)
	if v, ok := linkedTxCache.Get(bh); ok {
		bytes, ok := v.([]byte)
		if !ok {
			return nil, fmt.Errorf("expected []byte but got %T", v)
		}
		return bytes, nil
	}
	txID, err := ids.ToID(b)
	if err != nil {
		return nil, err
	}
	vk := PrefixTxValueKey(txID)
	v, err := db.Get(vk)
	if err != nil {
		return nil, err
	}
	linkedTxCache.Put(bh, v)
	return v, nil
}

func GetBalance(db database.KeyValueReader, address common.Address) (uint64, error) {
	k := PrefixBalanceKey(address)
	v, err := db.Get(k)
	if errors.Is(err, database.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(v), nil
}

func SetBalance(db database.KeyValueWriter, address common.Address, bal uint64) error {
	k := PrefixBalanceKey(address)
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, bal)
	return db.Put(k, b)
}

func ModifyBalance(db database.KeyValueReaderWriter, address common.Address, add bool, change uint64) (uint64, error) {
	b, err := GetBalance(db, address)
	if err != nil {
		return 0, err
	}
	var (
		n     uint64
		xflow bool
	)
	if add {
		n, xflow = smath.SafeAdd(b, change)
	} else {
		n, xflow = smath.SafeSub(b, change)
	}
	if xflow {
		return 0, fmt.Errorf("%w: bal=%d, addr=%v, add=%t, prev=%d, change=%d", ErrInvalidBalance, b, address, add, b, change)
	}
	return n, SetBalance(db, address, n)
}

func ApplyReward(
	db database.Database, blkID ids.ID, txID ids.ID, sender common.Address, reward uint64,
) (common.Address, bool, error) {
	seed := [64]byte{}
	copy(seed[:], blkID[:])
	copy(seed[32:], txID[:])
	iterator := crypto.Keccak256(seed[:])

	startKey := SpaceInfoKey(iterator)
	baseKey := SpaceInfoKey(nil)
	cursor := db.NewIteratorWithStart(startKey)
	defer cursor.Release()
	for cursor.Next() {
		curKey := cursor.Key()
		if bytes.Compare(baseKey, curKey) < -1 { // startKey < curKey; continue search
			continue
		}
		if !bytes.Contains(curKey, baseKey) { // curKey does not contain base key; end search
			break
		}

		var i SpaceInfo
		if _, err := Unmarshal(cursor.Value(), &i); err != nil {
			return common.Address{}, false, err
		}
		space := string(curKey[2:])

		// Do not give sender their funds back
		if bytes.Equal(i.Owner[:], sender[:]) {
			log.Debug("skipping reward: same owner", "space", space, "owner", i.Owner)
			return common.Address{}, false, nil
		}

		// Distribute reward
		if _, err := ModifyBalance(db, i.Owner, true, reward); err != nil {
			return common.Address{}, false, err
		}

		log.Debug("rewarded space owner", "space", space, "owner", i.Owner, "amount", reward)
		return i.Owner, true, nil
	}

	// No reward applied
	log.Debug("skipping reward: no valid space")
	return common.Address{}, false, cursor.Error()
}

func GetAllOwned(db database.Database, owner common.Address) (spaces []string, err error) {
	baseKey := PrefixOwnedKey(owner, nil)
	cursor := db.NewIteratorWithStart(baseKey)
	defer cursor.Release()
	spaces = []string{}
	for cursor.Next() {
		curKey := cursor.Key()
		if bytes.Compare(baseKey, curKey) < -1 { // startKey < curKey; continue search
			continue
		}
		if !bytes.Contains(curKey, baseKey) { // curKey does not contain base key; end search
			break
		}

		spaces = append(spaces,
			// [ownedPrefix] + [delimiter] + [address] + [delimiter] + [space]
			string(curKey[2+common.AddressLength+1:]),
		)
	}
	return spaces, cursor.Error()
}

func CompactablePrefixKey(pfx byte) []byte {
	return []byte{pfx, parser.ByteDelimiter}
}
