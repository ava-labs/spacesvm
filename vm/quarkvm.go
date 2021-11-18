// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"

	"github.com/ava-labs/quarkvm/crypto/ed25519"
	"github.com/ava-labs/quarkvm/storage"
)

const Name = "quarkvm"

// Service is the API service for this VM
type Service struct {
	vm *VM
}

type PutArgs struct {
	PublicKey ed25519.PublicKey `serialize:"true" json:"publicKey"`
	Signature string            `serialize:"true" json:"signature"`
	Key       string            `serialize:"true" json:"key"`
	Value     string            `serialize:"true" json:"value"`
}

type PutReply struct {
	Success bool `serialize:"true" json:"success"`
}

func (vm *VM) Put(args *PutArgs) error {
	if args.PublicKey == nil {
		return errors.New("the caller must provide a public key")
	}

	// TODO: check PoW
	return vm.s.Put(
		[]byte(args.Key),
		[]byte(args.Value),
		storage.WithOverwrite(true),
		storage.WithSignature(args.PublicKey, []byte(args.Signature)),
	)
}
