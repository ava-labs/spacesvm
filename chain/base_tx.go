package chain

import (
	"bytes"
	"errors"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/crypto"
	"github.com/ava-labs/quarkvm/types"
)

func init() {
	codec.RegisterType(&BaseTx{})
}

const (
	MaxPrefixSize = 256
)

func VerifyPrefixKey(prefix []byte) error {
	if len(prefix) == 0 {
		return errors.New("prefix cannot be empty")
	}
	if len(prefix) > MaxPrefixSize {
		return errors.New("prefix too big")
	}
	if bytes.IndexRune(prefix, types.PrefixDelimiter) != -1 {
		return errors.New("prefix contains delimiter")
	}
	return nil
}

type BaseTx struct {
	Sender   *crypto.PublicKey `serialize:"true"`
	Graffiti []byte            `serialize:"true"`
	BlockID  ids.ID            `serialize:"true"`
	Prefix   []byte            `serialize:"true"`
}

func (b *BaseTx) SetBlockID(blockID ids.ID) {
	b.BlockID = blockID
}

func (b *BaseTx) SetGraffiti(graffiti []byte) {
	b.Graffiti = graffiti
}

func (b *BaseTx) GetBlockID() ids.ID {
	return b.BlockID
}

func (b *BaseTx) GetSender() *crypto.PublicKey {
	return b.Sender
}

func (b *BaseTx) VerifyBase() error {
	if err := VerifyPrefixKey(b.Prefix); err != nil {
		return err
	}
	if b.Sender == nil {
		return errors.New("invalid sender")
	}
	if b.BlockID == ids.Empty {
		return errors.New("invalid blockID")
	}
	return nil
}
