// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"encoding/hex"

	"github.com/ava-labs/avalanchego/database"
	log "github.com/inconshreveable/log15"
)

var _ database.KeyValueWriterDeleter = &replayDB{}

type replayDB struct{}

func bytes2Hex(bytes []byte) string {
	return hex.EncodeToString(bytes)
}

// Put implements database.KeyValueWriterDeleter
func (*replayDB) Put(key []byte, value []byte) error {
	log.Info("put", "key", bytes2Hex(key), "val", bytes2Hex(value))
	return nil
}

// Delete implements database.KeyValueWriterDeleter
func (*replayDB) Delete(key []byte) error {
	log.Info("delete", "key", bytes2Hex(key))
	return nil
}
