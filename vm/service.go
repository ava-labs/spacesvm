// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"net/http"

	"github.com/ava-labs/avalanchego/ids"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/quarkvm/chain"
)

var (
	ErrPoWFailed      = errors.New("PoW failed")
	ErrInvalidEmptyTx = errors.New("invalid empty transaction")
)

type Service struct {
	vm *VM
}

type PingArgs struct {
}

type PingReply struct {
	Success bool `serialize:"true" json:"success"`
}

func (svc *Service) Ping(_ *http.Request, args *PingArgs, reply *PingReply) (err error) {
	log.Info("ping")
	reply.Success = true
	return nil
}

type IssueTxArgs struct {
	*chain.Transaction `serialize:"true" json:"tx"`
}

type IssueTxReply struct {
	TxID    string `serialize:"true" json:"txID"`
	Error   error  `serialize:"true" json:"error"`
	Success bool   `serialize:"true" json:"success"`
}

func (svc *Service) IssueTx(_ *http.Request, args *IssueTxArgs, reply *IssueTxReply) error {
	return nil
}

type CheckTxArgs struct {
	TxID string `serialize:"true" json:"txID"`
}

type CheckTxReply struct {
	TxID      ids.ID `serialize:"true" json:"txID"`
	Error     error  `serialize:"true" json:"error"`
	Confirmed bool   `serialize:"true" json:"confirmed"`
}

func (svc *Service) CheckTx(_ *http.Request, args *CheckTxArgs, reply *CheckTxReply) error {
	return nil
}
