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
	i, has, err := GetPrefixInfo(db, l.Prefix)
	if err != nil {
		return err
	}
	if !has {
		return errors.New("cannot add lifeline to missing tx")
	}
	// If you are "in debt", lifeline only adds but doesn't reset to new
	i.Expiry += expiryTime / i.Keys
	return PutPrefixInfo(db, l.Prefix, i)
}
