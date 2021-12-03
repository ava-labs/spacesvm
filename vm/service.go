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
	Tx *chain.Transaction `serialize:"true" json:"tx"`
}

type IssueTxReply struct {
	TxID    ids.ID `serialize:"true" json:"txID"`
	Error   error  `serialize:"true" json:"error"`
	Success bool   `serialize:"true" json:"success"`
}

func (svc *Service) IssueTx(_ *http.Request, args *IssueTxArgs, reply *IssueTxReply) error {
	svc.vm.Submit(args.Tx)
	reply.TxID = args.Tx.ID()
	reply.Success = true
	return nil
}

type CheckTxArgs struct {
	TxID ids.ID `serialize:"true" json:"txID"`
}

type CheckTxReply struct {
	Error     error `serialize:"true" json:"error"`
	Confirmed bool  `serialize:"true" json:"confirmed"`
}

func (svc *Service) CheckTx(_ *http.Request, args *CheckTxArgs, reply *CheckTxReply) error {
	has, err := chain.HasTransaction(svc.vm.db, args.TxID)
	if err != nil {
		reply.Error = err
		return nil
	}
	reply.Confirmed = has
	return nil
}

type CurrBlockArgs struct {
}

type CurrBlockReply struct {
	BlockID ids.ID `serialize:"true" json:"blockID"`
}

func (svc *Service) CurrBlock(_ *http.Request, args *CurrBlockArgs, reply *CurrBlockReply) error {
	reply.BlockID = svc.vm.preferred
	return nil
}

type ValidBlockIDArgs struct {
	BlockID ids.ID `serialize:"true" json:"blockID"`
}

type ValidBlockIDReply struct {
	Valid bool `serialize:"true" json:"valid"`
}

func (svc *Service) ValidBlockID(_ *http.Request, args *ValidBlockIDArgs, reply *ValidBlockIDReply) error {
	reply.Valid = svc.vm.ValidBlockID(args.BlockID)
	return nil
}

type DifficultyEstimateArgs struct {
}

type DifficultyEstimateReply struct {
	Difficulty uint64 `serialize:"true" json:"valid"`
}

func (svc *Service) DifficultyEstimate(_ *http.Request, args *DifficultyEstimateArgs, reply *DifficultyEstimateReply) error {
	reply.Difficulty = svc.vm.DifficultyEstimate()
	return nil
}
