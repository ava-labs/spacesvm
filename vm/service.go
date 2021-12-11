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

type PingArgs struct{}

type PingReply struct {
	Success bool `serialize:"true" json:"success"`
}

func (svc *Service) Ping(_ *http.Request, args *PingArgs, reply *PingReply) (err error) {
	log.Info("ping")
	reply.Success = true
	return nil
}

type IssueTxArgs struct {
	Tx []byte `serialize:"true" json:"tx"`
}

type IssueTxReply struct {
	TxID    ids.ID `serialize:"true" json:"txId"`
	Success bool   `serialize:"true" json:"success"`
}

func (svc *Service) IssueTx(_ *http.Request, args *IssueTxArgs, reply *IssueTxReply) error {
	tx := new(chain.Transaction)
	if _, err := chain.Unmarshal(args.Tx, tx); err != nil {
		return err
	}

	// otherwise, unexported tx.id field is empty
	if err := tx.Init(); err != nil {
		reply.Success = false
		return err
	}
	reply.TxID = tx.ID()

	err := svc.vm.Submit(tx)
	reply.Success = err == nil
	return err
}

type CheckTxArgs struct {
	TxID ids.ID `serialize:"true" json:"txId"`
}

type CheckTxReply struct {
	Confirmed bool `serialize:"true" json:"confirmed"`
}

func (svc *Service) CheckTx(_ *http.Request, args *CheckTxArgs, reply *CheckTxReply) error {
	has, err := chain.HasTransaction(svc.vm.db, args.TxID)
	if err != nil {
		return err
	}
	reply.Confirmed = has
	return nil
}

type CurrBlockArgs struct{}

type CurrBlockReply struct {
	BlockID ids.ID `serialize:"true" json:"blockId"`
}

func (svc *Service) CurrBlock(_ *http.Request, args *CurrBlockArgs, reply *CurrBlockReply) error {
	reply.BlockID = svc.vm.preferred
	return nil
}

type ValidBlockIDArgs struct {
	BlockID ids.ID `serialize:"true" json:"blockId"`
}

type ValidBlockIDReply struct {
	Valid bool `serialize:"true" json:"valid"`
}

func (svc *Service) ValidBlockID(_ *http.Request, args *ValidBlockIDArgs, reply *ValidBlockIDReply) error {
	valid, err := svc.vm.ValidBlockID(args.BlockID)
	if err != nil {
		return err
	}
	reply.Valid = valid
	return nil
}

type DifficultyEstimateArgs struct{}

type DifficultyEstimateReply struct {
	Difficulty uint64 `serialize:"true" json:"valid"`
}

func (svc *Service) DifficultyEstimate(
	_ *http.Request,
	_ *DifficultyEstimateArgs,
	reply *DifficultyEstimateReply,
) error {
	diff, err := svc.vm.DifficultyEstimate()
	if err != nil {
		return err
	}
	reply.Difficulty = diff
	return nil
}

type PrefixInfoArgs struct {
	Prefix []byte `serialize:"true" json:"prefix"`
}

type PrefixInfoReply struct {
	Info *chain.PrefixInfo `serialize:"true" json:"info"`
}

func (svc *Service) PrefixInfo(_ *http.Request, args *PrefixInfoArgs, reply *PrefixInfoReply) error {
	i, _, err := chain.GetPrefixInfo(svc.vm.db, args.Prefix)
	if err != nil {
		return err
	}
	reply.Info = i
	return nil
}

type RangeArgs struct {
	// Prefix is the namespace for the "PrefixInfo"
	// whose owner can write and read value for the
	// specific key space.
	// Assume the client pre-processes the inputs so that
	// all prefix must have the delimiter '/' as suffix.
	Prefix []byte `serialize:"true" json:"prefix"`

	// Key is parsed from the given input, with its prefix removed.
	// Optional for claim/lifeline transactions.
	// Non-empty to claim a key-value pair.
	Key []byte `serialize:"true" json:"key"`

	// RangeEnd is optional, and only non-empty for range query transactions.
	RangeEnd []byte `serialize:"true" json:"rangeEnd"`

	// Limit limits the number of key-value pairs in the response.
	Limit uint32 `serialize:"true" json:"limit"`
}

type RangeReply struct {
	KeyValues []chain.KeyValue `serialize:"true" json:"keyValues"`
	Error     error            `serialize:"true" json:"error"`
}

func (svc *Service) Range(_ *http.Request, args *RangeArgs, reply *RangeReply) (err error) {
	log.Debug("range query", "key", string(args.Key), "rangeEnd", string(args.RangeEnd))
	opts := make([]chain.OpOption, 0)
	if len(args.RangeEnd) > 0 {
		opts = append(opts, chain.WithRangeEnd(args.RangeEnd))
	}
	if args.Limit > 0 {
		opts = append(opts, chain.WithRangeLimit(args.Limit))
	}
	reply.KeyValues = chain.Range(svc.vm.db, args.Prefix, args.Key, opts...)
	reply.Error = nil
	return nil
}
