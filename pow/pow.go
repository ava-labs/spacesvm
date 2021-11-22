// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package pow defines Proof-of-Work mechanisms for VM admission control.
package pow

import (
	"encoding/json"
	"math/big"

	"ekyu.moe/cryptonight"
)

// Unit of work.
type Unit interface {
	// Enumerate to the next input.
	// Returns false if there's no next input available.
	Next() bool
	// Returns true if PoW is completed.
	Prove(difficulty uint64) bool
}

// New returns a new unit of PoW using "cryptonight".
func NewUnit(d []byte) Unit {
	u := &unit{Data: d, nonce: big.NewInt(0)}
	u.Nonce = u.nonce.Bytes()
	return u
}

type unit struct {
	Data  []byte   `json:"data"`
	Nonce []byte   `json:"nonce"`
	nonce *big.Int `json:"-"`
}

func (u *unit) Next() bool {
	u.Nonce = u.nonce.Add(u.nonce, big.NewInt(1)).Bytes()
	return true
}

func (u *unit) Prove(difficulty uint64) bool {
	output, err := json.Marshal(u)
	if err != nil {
		panic(err)
	}
	hash := cryptonight.Sum(output, 2)
	// returns true if hash > difficulty
	proved := cryptonight.CheckHash(hash, difficulty)
	return proved
}
