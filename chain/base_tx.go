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
	// TODO: change types
	Sender   *crypto.PublicKey `serialize:"true"`
	Graffiti []byte            `serialize:"true"`
	BlockID  ids.ID            `serialize:"true"`
	Prefix   []byte            `serialize:"true"`
}

// TODO: need public setters?
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
		return ErrInvalidSender
	}
	if b.BlockID == ids.Empty {
		return ErrInvalidBlockID
	}
	return nil
}
