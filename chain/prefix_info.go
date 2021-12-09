package chain

import (
	"github.com/ava-labs/quarkvm/crypto"
)

type PrefixInfo struct {
	Owner       [crypto.PublicKeySize]byte `serialize:"true"`
	LastUpdated int64                      `serialize:"true"`
	Expiry      int64                      `serialize:"true"`
	Keys        int64                      `serialize:"true"` // decays faster the more keys you have
}
