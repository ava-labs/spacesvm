// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package quarkvm

import "github.com/ava-labs/avalanchego/ids"

// memPool defines in-memory transaction pool.
type memPool interface {
	push(tx transaction)
	peekMax() (transaction, uint64)
	peekMin() (transaction, uint64)
	popMax() (transaction, uint64)
	popMin() transaction
	remove(id ids.ID) transaction
	prune(validHashes ids.Set)
	len() int
	get(id ids.ID) (transaction, bool)
	has(id ids.ID) bool
}
