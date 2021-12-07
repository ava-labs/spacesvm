package chain

type PrefixInfo struct {
	Owner       string `serialize:"true"`
	LastUpdated int64  `serialize:"true"`
	Expiry      int64  `serialize:"true"`
	Keys        int64  `serialize:"true"` // decays faster the more keys you have
}
