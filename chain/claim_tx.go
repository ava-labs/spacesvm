package chain

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ava-labs/avalanchego/utils/formatting"

	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/types"
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
	i, err := db.GetPrefixInfo(c.Prefix)
	if err != nil {
		return err
	}
	if i.Expiry >= blockTime {
		return fmt.Errorf("prefix %s not expired", c.Prefix)
	}
	return c.accept(db, blockTime)
}

func (c *ClaimTx) accept(db DB, blockTime int64) error {
	i := &types.PrefixInfo{Owner: c.Sender, LastUpdated: blockTime, Expiry: blockTime + expiryTime, Keys: 1}
	if err := db.PutPrefixInfo(c.Prefix, i); err != nil {
		return err
	}
	// Remove anything that is stored in value prefix
	return db.DeleteAllPrefixKeys(c.Prefix)
}
