package types

import (
	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/crypto"
)

func init() {
	codec.RegisterType(&PrefixInfo{})
}

type PrefixInfo struct {
	Owner       *crypto.PublicKey `serialize:"true"`
	LastUpdated int64             `serialize:"true"`
	Expiry      int64             `serialize:"true"`
	Keys        int64             `serialize:"true"` // decays faster the more keys you have
}
