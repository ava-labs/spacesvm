// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"

	"github.com/ava-labs/quarkvm/parser"
)

func Reward(g *Genesis, db database.KeyValueReaderWriter, prefix []byte) error {
	// If there is no one to reward, do nothing
	if prefix == nil {
		return nil
	}
	if err := parser.CheckPrefix(prefix); err != nil {
		return err
	}
	return addLife(g, db, prefix)
}
