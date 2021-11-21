// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package codec imports default message codec managers.
package codec

import (
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/codec/linearcodec"
)

const (
	// CodecVersion is the current default codec version
	codecVersion = 0
)

var (
	// Codecs do serialization and deserialization
	codecManager codec.Manager
	c            linearcodec.Codec
)

func init() {
	c = linearcodec.NewDefault()
	codecManager = codec.NewDefaultManager()

	if err := codecManager.RegisterCodec(codecVersion, c); err != nil {
		panic(err)
	}
}

// Manager returns the initialized codec manager.
func Manager() codec.Manager {
	return codecManager
}

func Marshal(source interface{}) ([]byte, error) {
	return codecManager.Marshal(codecVersion, source)
}

func Unmarshal(source []byte, destination interface{}) (uint16, error) {
	return codecManager.Unmarshal(source, destination)
}

func RegisterType(t interface{}) {
	c.RegisterType(t)
}
