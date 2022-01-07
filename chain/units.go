// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

// Units is the "cost" of a value
func Units(b []byte) int64 {
	return int64(len(b)/ValueUnitLength + 1)
}
