package chain

import (
	"bytes"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/quarkvm/crypto"
)

func VerifyPrefixKey(prefix []byte) error {
	if len(prefix) == 0 {
		return ErrPrefixEmpty
	}
	if len(prefix) > MaxPrefixSize {
		return ErrPrefixTooBig
	}
	if bytes.IndexRune(prefix, PrefixDelimiter) != -1 {
		return ErrPrefixContainsDelim
	}
	return nil
}

type BaseTx struct {
	Sender   [crypto.PublicKeySize]byte `serialize:"true"`
	Graffiti uint64                     `serialize:"true"`
	BlockID  ids.ID                     `serialize:"true"`
	Prefix   []byte                     `serialize:"true"`
}

func (b *BaseTx) SetBlockID(blockID ids.ID) {
	b.BlockID = blockID
}

func (b *BaseTx) SetGraffiti(graffiti uint64) {
	b.Graffiti = graffiti
}

func (b *BaseTx) GetBlockID() ids.ID {
	return b.BlockID
}

func (b *BaseTx) GetSender() [crypto.PublicKeySize]byte {
	return b.Sender
}

func (b *BaseTx) ExecuteBase() error {
	if err := VerifyPrefixKey(b.Prefix); err != nil {
		return err
	}
	if len(b.Sender) == 0 {
		return ErrInvalidSender
	}
	if b.BlockID == ids.Empty {
		return ErrInvalidBlockID
	}
	return nil
}
