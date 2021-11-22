// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/prefixdb"
	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/crypto"
	"github.com/ava-labs/quarkvm/owner"
)

var (
	blockStateBucket = []byte("block")
	ownerBucket      = []byte("owner")
	txBucket         = []byte("tx")
	keyBucket        = []byte("key")
)

var (
	ErrNoPubKey           = errors.New("caller must provide the public key to make modification or overwrite")
	ErrPubKeyNotAllowed   = errors.New("public key is not allowed for this operation")
	ErrKeyExists          = errors.New("key already exists")
	ErrKeyNotExist        = errors.New("key not exists")
	ErrInvalidKeyLength   = errors.New("invalid key length")
	ErrInvalidValueLength = errors.New("invalid value length")
)

const (
	maxKeyLength   = 256
	maxValueLength = 1024 // limit to 1 KiB for now
)

type Storage interface {
	Block() database.Database
	Owner() database.Database
	Tx() database.Database
	Key() database.Database
	Commit() error
	Close() error

	// Finds the underlying info based on the key.
	// The method assumes the prefix is already pre-processed.
	FindOwner(k []byte) (ov *owner.Owner, err error)
	Put(k []byte, v []byte, opts ...OpOption) error
	Range(k []byte, opts ...OpOption) (resp *RangeResponse, err error)
}

type storage struct {
	ctx     *snow.Context
	baseDB  *versiondb.Database
	blockDB *prefixdb.Database
	ownerDB *prefixdb.Database
	txDB    *prefixdb.Database
	keyDB   *prefixdb.Database
}

func New(ctx *snow.Context, db database.Database) Storage {
	baseDB := versiondb.New(db)
	return &storage{
		ctx:     ctx,
		baseDB:  baseDB,
		blockDB: prefixdb.New(blockStateBucket, baseDB),
		ownerDB: prefixdb.New(ownerBucket, baseDB),
		txDB:    prefixdb.New(txBucket, baseDB),
		keyDB:   prefixdb.New(keyBucket, baseDB),
	}
}

func (s *storage) Block() database.Database {
	return s.blockDB
}

func (s *storage) Owner() database.Database {
	return s.ownerDB
}

func (s *storage) Tx() database.Database {
	return s.txDB
}

func (s *storage) Key() database.Database {
	return s.keyDB
}

func (s *storage) Commit() error {
	return s.baseDB.Commit()
}

func (s *storage) Close() error {
	return s.baseDB.Close()
}

func (s *storage) FindOwner(k []byte) (ov *owner.Owner, err error) {
	// TODO: can we do this in one db call (e.g., Get)?
	exist, err := s.ownerDB.Has(k)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, ErrKeyNotExist
	}

	src, err := s.ownerDB.Get(k)
	if err != nil {
		return nil, err
	}

	ov = new(owner.Owner)
	if _, err := codec.Unmarshal(src, ov); err != nil {
		return nil, err
	}
	return ov, nil
}

// TODO: version control?

func (s *storage) Put(k []byte, v []byte, opts ...OpOption) error {
	if len(k) > maxKeyLength || len(k) == 0 {
		return ErrInvalidKeyLength
	}
	if len(v) > maxValueLength {
		return ErrInvalidValueLength
	}

	ret := &Op{key: k}
	ret.applyOpts(opts)
	if ret.pub == nil {
		return ErrNoPubKey
	}

	if exist, _ := s.keyDB.Has(k); exist && !ret.overwrite {
		return ErrKeyExists
	}
	pfx, _, err := GetPrefix(k)
	if err != nil {
		return err
	}

	// check the ownership of the key
	// any non-existent/expired key can be claimed by anyone
	// that submits a sufficient PoW
	prevOwner, err := s.FindOwner(pfx)
	if err != nil {
		if err != ErrKeyNotExist {
			return err
		}
		prevOwner = nil // make sure no previous owner is set
	}

	needNewOwner := true
	if prevOwner != nil { // prefix previously claimed
		expired := prevOwner.Expiry < time.Now().Unix()
		sameOwner := bytes.Equal(prevOwner.PublicKey, ret.pub.Bytes())
		switch {
		case !expired && !sameOwner:
			return fmt.Errorf("namespace %q has not been expired and already owned by someone else", prevOwner.Namespace)
		case !expired && sameOwner:
			needNewOwner = false
			s.ctx.Log.Info("%q has an active owner", prevOwner.Namespace)
		case expired:
			s.ctx.Log.Info("%q has an expired owner, allowing new owner claim", prevOwner.Namespace)
		}
	}
	newOwner := prevOwner
	if needNewOwner { // prefix expired or not claimed yet
		newOwner = &owner.Owner{
			PublicKey: ret.pub.Bytes(),
			Namespace: string(pfx),
		}
	}

	// refresh update timestamps on put
	lastUpdated := time.Now()
	newOwner.LastUpdated = lastUpdated.Unix()

	// decays faster the more keys you have
	expiry := time.Hour
	if prevOwner != nil {
		expiry = time.Duration((prevOwner.Expiry - prevOwner.LastUpdated) * prevOwner.Keys)
	}
	newOwner.Keys++
	expiry /= time.Duration(newOwner.Keys)

	newOwner.Expiry = lastUpdated.Add(expiry).Unix()

	newOwnerBytes, err := codec.Marshal(newOwner)
	if err != nil {
		return nil
	}
	if err := s.ownerDB.Put(pfx, newOwnerBytes); err != nil {
		return err
	}

	// if validated or new key,
	// the owner is allowed to write the key
	return s.keyDB.Put(k, v)
}

type RangeResponse struct {
	KeyValues []KeyValue `json:"keyValues"`
}

type KeyValue struct {
	Key   []byte `json:"key"`
	Value []byte `json:"value"`
}

func (s *storage) Range(k []byte, opts ...OpOption) (resp *RangeResponse, err error) {
	if len(k) > maxKeyLength || len(k) == 0 {
		return nil, ErrInvalidKeyLength
	}

	ret := &Op{key: k, rangeLimit: 10}
	ret.applyOpts(opts)
	if ret.pub == nil {
		return nil, ErrNoPubKey
	}

	pfx, endKey, err := GetPrefix(k)
	if err != nil {
		return nil, err
	}

	// just check ownership with prefix
	prevOwner, err := s.FindOwner(pfx)
	if err != nil {
		return nil, err
	}
	sameOwner := bytes.Equal(prevOwner.PublicKey, ret.pub.Bytes())
	if !sameOwner {
		// TODO: should we allow for public reads? or limit the range?
		return nil, ErrPubKeyNotAllowed
	}

	if len(ret.endKey) > 0 {
		endKey = ret.endKey
	}

	resp = new(RangeResponse)
	cursor := s.keyDB.NewIteratorWithStart(k)
	for cursor.Next() {
		cur := cursor.Key()
		if bytes.Compare(endKey, cur) <= 0 { // endKey <= cur
			break
		}
		resp.KeyValues = append(resp.KeyValues, KeyValue{Key: cur, Value: cursor.Value()})
		if ret.rangeLimit > 0 && len(resp.KeyValues) == ret.rangeLimit {
			break
		}
	}
	return resp, nil
}

type Op struct {
	overwrite bool
	blockTime int64
	pub       crypto.PublicKey

	key    []byte
	endKey []byte

	// TODO: make this configurable
	rangeLimit int
}

type OpOption func(*Op)

func (op *Op) applyOpts(opts []OpOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func WithOverwrite(b bool) OpOption {
	return func(op *Op) { op.overwrite = b }
}

func WithBlockTime(t int64) OpOption {
	return func(op *Op) { op.blockTime = t }
}

func WithPublicKey(pub crypto.PublicKey) OpOption {
	return func(op *Op) {
		op.pub = pub
	}
}

func WithPrefix() OpOption {
	return func(op *Op) {
		_, op.endKey, _ = GetPrefix(op.key)
	}
}

func WithRangeEnd(end string) OpOption {
	return func(op *Op) { op.endKey = []byte(end) }
}
