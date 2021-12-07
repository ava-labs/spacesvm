package chain

import (
	"bytes"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/utils/formatting"
)

var (
	_ UnsignedTransaction = &ClaimTx{}
)

type ClaimTx struct {
	*BaseTx `serialize:"true"`
}

func (c *ClaimTx) Verify(db database.Database, blockTime int64) error {
	// Restrict address prefix to be owned by pk
	if decodedPrefix, err := formatting.Decode(formatting.CB58, string(c.Prefix)); err == nil {
		if !bytes.Equal(c.Sender.Bytes(), decodedPrefix) {
			return ErrPublicKeyMismatch
		}
	}
	i, has, err := GetPrefixInfo(db, c.Prefix)
	if err != nil {
		return err
	}
	if !has {
		return c.accept(db, blockTime)
	}
	if i.Expiry >= blockTime {
		return ErrPrefixNotExpired
	}
	return c.accept(db, blockTime)
}

func (c *ClaimTx) accept(db database.Database, blockTime int64) error {
	i := &PrefixInfo{Owner: c.Sender.Address(), LastUpdated: blockTime, Expiry: blockTime + expiryTime, Keys: 1}
	if err := PutPrefixInfo(db, c.Prefix, i); err != nil {
		return err
	}
	// Remove anything that is stored in value prefix
	return DeleteAllPrefixKeys(db, c.Prefix)
}
