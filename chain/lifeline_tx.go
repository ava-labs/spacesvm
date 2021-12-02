package chain

import (
	"errors"

	"github.com/ava-labs/quarkvm/codec"
)

func init() {
	codec.RegisterType(&LifelineTx{})
}

var (
	_ UnsignedTransaction = &LifelineTx{}
)

type LifelineTx struct {
	*BaseTx `serialize:"true"`
}

func (l *LifelineTx) Verify(db DB, blockTime int64) error {
	has, err := db.HasPrefix(l.Prefix)
	if err != nil {
		return err
	}
	if !has {
		return errors.New("cannot add lifeline to missing tx")
	}
	// Anyone can choose to support a prefix (not just owner)
	i, err := db.GetPrefixInfo(l.Prefix)
	if err != nil {
		return err
	}
	// If you are "in debt", lifeline only adds but doesn't reset to new
	i.Expiry += expiryTime / i.Keys
	return db.PutPrefixInfo(l.Prefix, i)
}
