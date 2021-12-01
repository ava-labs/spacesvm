package chain

import (
	"bytes"
	"errors"

	"github.com/ava-labs/avalanchego/database"
)

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
	k := PrefixInfoKey(s.Prefix)
	has, err := db.Has(k)
	if err != nil {
		return err
	}
	if !has {
		return errors.New("cannot set if prefix doesn't exist")
	}
	v, err := db.Get(k)
	if err != nil {
		return err
	}
	var i PrefixInfo
	if _, err := codecManager.Unmarshal(v, &i); err != nil {
		return err
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
	has, err = db.Has(PrefixValueKey(s.Prefix, s.Key))
	if err != nil {
		return err
	}
	if !has {
		return errors.New("cannot delete non-existent key")
	}
	return s.accept(db, blockTime)
}

func (s *SetTx) accept(db database.Database, blockTime int64) error {
	k := PrefixInfoKey(s.Prefix)
	v, err := db.Get(k)
	if err != nil {
		return err
	}
	var i PrefixInfo
	if _, err := codecManager.Unmarshal(v, &i); err != nil {
		return err
	}
	kv := PrefixValueKey(s.Prefix, s.Key)
	timeRemaining := (i.Expiry - i.LastUpdated) * i.Keys
	if len(s.Value) == 0 {
		i.Keys--
		if err := db.Delete(kv); err != nil {
			return err
		}
	} else {
		i.Keys++
		if err := db.Put(kv, s.Value); err != nil {
			return err
		}
	}
	newTimeRemaining := timeRemaining / i.Keys
	i.LastUpdated = blockTime
	i.Expiry = blockTime + newTimeRemaining
	b, err := codecManager.Marshal(codecVersion, i)
	if err != nil {
		return err
	}
	return db.Put(k, b)
}
