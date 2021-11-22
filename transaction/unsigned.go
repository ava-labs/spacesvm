// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package transaction

import (
	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/storage"
)

func init() {
	codec.RegisterType(&Unsigned{})
}

type Unsigned struct {
	PublicKey []byte `serialize:"true" json:"publicKey"`

	// Op is either Put or Range
	// TODO: Delete, DeleteRange
	Op       string `serialize:"true" json:"op"`
	Key      string `serialize:"true" json:"key"`
	Value    string `serialize:"true" json:"value,omitempty"`
	RangeEnd string `serialize:"true" json:"rangeEnd,omitempty"`
}

func (utx Unsigned) Bytes() []byte {
	v, err := codec.Marshal(utx)
	if err != nil {
		panic(err)
	}
	return v
}

func (utx Unsigned) GetPrefix() ([]byte, error) {
	pfx, _, err := storage.GetPrefix([]byte(utx.Key))
	return pfx, err
}
