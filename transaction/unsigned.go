// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package transaction

import (
	"errors"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/crypto/ed25519"
	"github.com/ava-labs/quarkvm/storage"
)

var (
	ErrPrefixNotExist      = errors.New("prefix does not exist")
	ErrPrefixOwnerMismatch = errors.New("prefix owner mismatch")
	ErrPrefixNotExpired    = errors.New("prefix not expired")
	ErrPrefixExpired       = errors.New("prefix expired")
	ErrKeyNotExist         = errors.New("key does not exist")
)

type Unsigned interface {
	GetBlockID() ids.ID
	SetBlockID(block ids.ID)
	SetGraffiti(graffiti []byte)
	GetSender() ed25519.PublicKey
	GetPrefix() []byte
	VerifyBase() error
	Verify(storage.Storage, int64) error
	Accept(storage.Storage, int64) error
}

const maxKeyLength = 256

func init() {
	codec.RegisterType(&ed25519.PublicKeyED25519{})
	codec.RegisterType(&base{})
}

type base struct {
	Sender   ed25519.PublicKey `serialize:"true"`
	Graffiti []byte            `serialize:"true"`
	BlockID  ids.ID            `serialize:"true"`
	Prefix   []byte            `serialize:"true"`
}

func (b *base) SetBlockID(blockID ids.ID) {
	b.BlockID = blockID
}

func (b *base) SetGraffiti(graffiti []byte) {
	b.Graffiti = graffiti
}

func (b *base) GetBlockID() ids.ID {
	return b.BlockID
}

func (b *base) GetSender() ed25519.PublicKey {
	return b.Sender
}

func (b *base) GetPrefix() []byte {
	return b.Prefix
}

func (b *base) VerifyBase() error {
	if len(b.Prefix) > maxKeyLength || len(b.Prefix) == 0 {
		return errors.New("invalid length")
	}
	if b.Sender == nil {
		return errors.New("invalid sender")
	}
	if b.BlockID == ids.Empty {
		return errors.New("invalid blockID")
	}
	return nil
}
