package chain

import (
	"bytes"
	"errors"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/quarkvm/codec"
)

func init() {
	codec.RegisterType(&SetTx{})
}

var (
	_ UnsignedTransaction = &SetTx{}
)

type SetTx struct {
	*BaseTx `serialize:"true"`
	Key     []byte `serialize:"true"`
	Value   []byte `serialize:"true"`
}

func (s *SetTx) Verify(db database.Database, blockTime int64) error {
	if len(s.Key) > maxKeyLength || len(s.Key) == 0 {
		return errors.New("invalid key length")
	}
	if len(s.Value) > maxKeyLength {
		return errors.New("invalid value length")
	}
	i, has, err := GetPrefixInfo(db, s.Prefix)
	if err != nil {
		return err
	}
	if !has {
		return errors.New("cannot set if prefix doesn't exist")
	}
	if !bytes.Equal(i.Owner.Bytes(), s.Sender.Bytes()) {
		return errors.New("prefix not owned by modifier")
	}
	if i.Expiry < blockTime {
		return errors.New("prefix expired")
	}
	// If we are trying to delete a key, make sure it previously exists.
	if len(s.Value) > 0 {
		return s.accept(db, blockTime)
	}
	has, err = HasPrefixKey(db, s.Prefix, s.Key)
	if err != nil {
		return err
	}
	if !has {
		return errors.New("cannot delete non-existent key")
	}
	return s.accept(db, blockTime)
}

func (s *SetTx) accept(db database.Database, blockTime int64) error {
	i, _, err := GetPrefixInfo(db, s.Prefix)
	if err != nil {
		return err
	}
	timeRemaining := (i.Expiry - i.LastUpdated) * i.Keys
	if len(s.Value) == 0 {
		i.Keys--
		if err := DeletePrefixKey(db, s.Prefix, s.Key); err != nil {
			return err
		}
	} else {
		i.Keys++
		if err := PutPrefixKey(db, s.Prefix, s.Key, s.Value); err != nil {
			return err
		}
	}
	newTimeRemaining := timeRemaining / i.Keys
	i.LastUpdated = blockTime
	i.Expiry = blockTime + newTimeRemaining
	return PutPrefixInfo(db, s.Prefix, i)
}
