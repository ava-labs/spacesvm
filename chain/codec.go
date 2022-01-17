// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/codec/linearcodec"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ava-labs/avalanchego/utils/wrappers"
)

const (
	// codecVersion is the current default codec version
	codecVersion = 0

	// maxSize is 4MB to support large values
	maxSize = 4 * units.MiB
)

var codecManager codec.Manager

func init() {
	c := linearcodec.NewDefault()
	codecManager = codec.NewManager(maxSize)
	errs := wrappers.Errs{}
	errs.Add(
		c.RegisterType(&BaseTx{}),
		c.RegisterType(&ClaimTx{}),
		c.RegisterType(&LifelineTx{}),
		c.RegisterType(&SetTx{}),
		c.RegisterType(&DeleteTx{}),
		c.RegisterType(&TransferTx{}),
		c.RegisterType(&MoveTx{}),
		c.RegisterType(&Transaction{}),
		c.RegisterType(&StatefulBlock{}),
		c.RegisterType(&SpaceInfo{}),
		c.RegisterType(&CustomAllocation{}),
		c.RegisterType(&Airdrop{}),
		c.RegisterType(&Genesis{}),
		codecManager.RegisterCodec(codecVersion, c),
	)
	if errs.Errored() {
		panic(errs.Err)
	}
}

func Marshal(source interface{}) ([]byte, error) {
	return codecManager.Marshal(codecVersion, source)
}

func Unmarshal(source []byte, destination interface{}) (uint16, error) {
	return codecManager.Unmarshal(source, destination)
}
