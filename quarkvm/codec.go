// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package quarkvm

import (
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/codec/linearcodec"
	"github.com/ava-labs/avalanchego/utils/wrappers"
)

const (
	// CodecVersion is the current default codec version
	codecVersion = 0
)

// Codecs do serialization and deserialization
var (
	Codec codec.Manager
)

func init() {
	c := linearcodec.NewDefault()
	Codec = codec.NewDefaultManager()

	errs := wrappers.Errs{}

	errs.Add(
		Codec.RegisterCodec(codecVersion, c),
	)
	if errs.Errored() {
		panic(errs.Err)
	}
}
