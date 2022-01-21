// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"encoding/json"
	"math/big"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
)

type SpaceInfo struct {
	Owner   common.Address `serialize:"true" json:"owner"`
	Created uint64         `serialize:"true" json:"created"`
	Updated uint64         `serialize:"true" json:"updated"`
	Expiry  uint64         `serialize:"true" json:"expiry"`
	Units   uint64         `serialize:"true" json:"units"` // decays faster the more units you have

	RawSpace  ids.ShortID `serialize:"true" json:"rawSpace"`
	Keys      []byte      `serialize:"true" json:"keys"`      // big.Int encoded bytes
	ValueSize []byte      `serialize:"true" json:"valueSize"` // big.Int encoded bytes
}

func (i *SpaceInfo) MarshalJSON() ([]byte, error) {
	type Alias SpaceInfo
	return json.Marshal(struct {
		Keys      string `json:"keys"`
		ValueSize string `json:"valueSize"`
	}{
		Keys:      new(big.Int).SetBytes(i.Keys).String(),
		ValueSize: new(big.Int).SetBytes(i.ValueSize).String(),
	})
}

func (i *SpaceInfo) UnmarshalJSON(b []byte) error {
	type Alias SpaceInfo
	r := struct {
		Keys      string `json:"keys"`
		ValueSize string `json:"valueSize"`
		*Alias
	}{
		Alias: (*Alias)(i),
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return err
	}
	k, ok := new(big.Int).SetString(r.Keys, 10)
	if !ok {
		return ErrNotANumber
	}
	i.Keys = k.Bytes()
	s, ok := new(big.Int).SetString(r.ValueSize, 10)
	if !ok {
		return ErrNotANumber
	}
	i.ValueSize = s.Bytes()
	return nil
}
