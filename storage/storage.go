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
	"github.com/ava-labs/quarkvm/crypto/ed25519"
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
	ErrInvalidSig         = errors.New("invalid signature")
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
	// The method should handle the prefix extraction.
	// Returns an error if the prefix is non-existent.
	FindOwner(k []byte) (prefix []byte, ov *owner.Owner, err error)
	Put(k []byte, v []byte, opts ...OpOption) error
	Get(k []byte, opts ...OpOption) ([]byte, bool, error)
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

func (s *storage) FindOwner(k []byte) (pfx []byte, ov *owner.Owner, err error) {
	pfx, _, err = getPrefix(k)
	if err != nil {
		return pfx, nil, err
	}

	// TODO: do this in one db call (e.g., Get)
	exist, err := s.ownerDB.Has(pfx)
	if err != nil {
		return pfx, nil, err
	}
	if !exist {
		return pfx, nil, ErrKeyNotExist
	}

	src, err := s.ownerDB.Get(pfx)
	if err != nil {
		return pfx, nil, err
	}

	ov = new(owner.Owner)
	if _, err := codec.Unmarshal(src, ov); err != nil {
		return pfx, nil, err
	}
	return pfx, ov, nil
}

func (s *storage) Put(k []byte, v []byte, opts ...OpOption) error {
	if len(k) > maxKeyLength || len(k) == 0 {
		return ErrInvalidKeyLength
	}
	if len(v) > maxValueLength {
		return ErrInvalidValueLength
	}

	ret := &Op{}
	ret.applyOpts(opts)
	if ret.pub == nil {
		return ErrNoPubKey
	}

	// value must be signe with signature
	if !ret.pub.Verify(v, ret.sig) {
		return ErrInvalidSig
	}

	if exist, _ := s.keyDB.Has(k); exist && !ret.overwrite {
		return ErrKeyExists
	}

	// check the ownership of the key
	// any non-existent/expired key can be claimed by anyone
	// that submits a sufficient PoW
	exists := true
	pfx, prevOwner, err := s.FindOwner(k)
	if err != nil {
		if err != ErrKeyNotExist {
			return err
		}
		exists = false
	}
	if exists && prevOwner == nil { // should never happen
		panic("key exists but owner not found?")
	}

	needNewOwner := true
	if exists {
		// prefix already claimed
		expired := prevOwner.Expiry < time.Now().Unix()
		sameOwner := bytes.Equal(prevOwner.PublicKey.Bytes(), ret.pub.Bytes())
		switch {
		case !expired && !sameOwner:
			return fmt.Errorf("%q is not expired and already owned by %q", prevOwner.Namespace, prevOwner.PublicKey.Address())
		case !expired && sameOwner:
			needNewOwner = false
			s.ctx.Log.Info("%q has an active owner", prevOwner.Namespace)
		case expired:
			s.ctx.Log.Info("%q has an expired owner; allowing put for new owner", prevOwner.Namespace)
		}
	}

	// prefix expired or not claimed yet
	newOwner := prevOwner
	if needNewOwner {
		newOwner = &owner.Owner{
			PublicKey: ret.pub,
			Namespace: string(pfx),
		}
	}

	// TODO: define save owner method
	// TODO: update other fields
	// TODO: make this configurable
	lastUpdated := time.Now()
	newOwner.LastUpdated = lastUpdated.Unix()
	newOwner.Expiry = lastUpdated.Add(time.Hour).Unix()
	newOwner.Keys++
	newOwnerBytes, err := codec.Marshal(newOwner)
	if err != nil {
		return nil
	}
	if err := s.ownerDB.Put(pfx, newOwnerBytes); err != nil {
		return err
	}

	// if validated or new key,
	// the owner is allowed to write the key
	// TODO: encrypt value
	return s.keyDB.Put(k, v)
}

func (s *storage) Get(k []byte, opts ...OpOption) ([]byte, bool, error) {
	ret := &Op{}
	ret.applyOpts(opts)
	if ret.pub == nil {
		return nil, false, ErrNoPubKey
	}

	pfx, _, err := getPrefix(k)
	if err != nil {
		return nil, false, err
	}
	_ = pfx

	// just check ownership with prefix
	// for now just return the value

	has, err := s.keyDB.Has(k)
	if err != nil {
		return nil, false, err
	}
	if !has {
		return nil, false, nil
	}
	v, err := s.keyDB.Get(k)
	if err != nil {
		return nil, false, err
	}
	return v, true, nil
}

type Op struct {
	overwrite bool
	pub       ed25519.PublicKey
	sig       []byte
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

func WithSignature(pub ed25519.PublicKey, sig []byte) OpOption {
	return func(op *Op) {
		op.pub = pub
		op.sig = sig
	}
}
