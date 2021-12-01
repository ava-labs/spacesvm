package chain

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/utils/formatting"

	"github.com/ava-labs/quarkvm/codec"
)

func init() {
	codec.RegisterType(&ClaimTx{})
}

var (
	_ UnsignedTransaction = &ClaimTx{}
)

type ClaimTx struct {
	*BaseTx `serialize:"true"`
}

func (c *ClaimTx) Verify(db DB, blockTime int64) error {
	// Restrict address prefix to be owned by pk
	if decodedPrefix, err := formatting.Decode(formatting.CB58, string(c.Prefix)); err == nil {
		if !bytes.Equal(c.Sender.Bytes(), decodedPrefix) {
			return errors.New("public key does not match decoded prefix")
		}
	}

	has, err := db.HasPrefix(c.Prefix)
	if err != nil {
		return err
	}
	if !has {
		return c.accept(db, blockTime)
	}
	v, err := db.Get(k)
	if err != nil {
		return err
	}
	var i PrefixInfo
	if _, err := codecManager.Unmarshal(v, &i); err != nil {
		return err
	}
	if i.Expiry >= blockTime {
		return fmt.Errorf("prefix %s not expired", c.Prefix)
	}
	return c.accept(db, blockTime)
}

func (c *ClaimTx) accept(db DB, blockTime int64) error {
	i := &PrefixInfo{Owner: c.Sender, LastUpdated: blockTime, Expiry: blockTime + expiryTime, Keys: 1}
	k := PrefixInfoKey(c.Prefix)
	b, err := codecManager.Marshal(codecVersion, i)
	if err != nil {
		return err
	}
	if err := db.Put(k, b); err != nil {
		return err
	}
	// Remove anything that is stored in value prefix
	return database.ClearPrefix(db, db, PrefixValueKey(c.Prefix, nil))
}
