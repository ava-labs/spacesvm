package chain

import (
	"errors"

	"github.com/ava-labs/avalanchego/database"
)

var (
	_ UnsignedTransaction = &LifelineTx{}
)

type LifelineTx struct {
	*BaseTx `serialize:"true"`
}

func (l *LifelineTx) Verify(db database.Database, blockTime int64) error {
	k := PrefixInfoKey(l.Prefix)
	has, err := db.Has(k)
	if err != nil {
		return err
	}
	if !has {
		return errors.New("cannot add lifeline to missing tx")
	}
	// Anyone can choose to support a prefix (not just owner)
	v, err := db.Get(k)
	if err != nil {
		return err
	}
	var i PrefixInfo
	if _, err := codecManager.Unmarshal(v, &i); err != nil {
		return err
	}
	// If you are "in debt", lifeline only adds but doesn't reset to new
	i.Expiry += expiryTime / i.Keys
	b, err := codecManager.Marshal(codecVersion, i)
	if err != nil {
		return err
	}
	return db.Put(k, b)
}
