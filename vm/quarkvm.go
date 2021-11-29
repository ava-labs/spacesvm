// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/quarkvm/crypto/ed25519"
	"github.com/ava-labs/quarkvm/pow"
	"github.com/ava-labs/quarkvm/storage"
	"github.com/ava-labs/quarkvm/transaction"
	log "github.com/inconshreveable/log15"
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
	// TODO: debug
	log.Info("ping")

	reply.Success = true
	return nil
}

type IssueTxArgs struct {
	*transaction.Transaction `serialize:"true" json:"tx"`
}

type IssueTxReply struct {
	RangeResponse *storage.RangeResponse `serialize:"true" json:"rangeResponse,omitempty"`
	TxID          string                 `serialize:"true" json:"txID"`
	Error         error                  `serialize:"true" json:"error"`
	Success       bool                   `serialize:"true" json:"success"`
}

func (svc *Service) IssueTx(_ *http.Request, args *IssueTxArgs, reply *IssueTxReply) (err error) {
	if args.Transaction == nil {
		reply.Error = ErrInvalidEmptyTx
		reply.Success = false
		return ErrInvalidEmptyTx
	}

	log.Debug("issuing transaction")

	// check before storage path, to prevent unnecessary mining
	// TODO: should transaction verify take storage to look up keys and previous owner?
	// otherwise, invalid Put will fail but still does PoW
	if err = args.Transaction.Verify(); err != nil {
		reply.Error = err
		reply.Success = false
		log.Debug("transaction failed verification", "error", err)
		return err
	}
	log.Debug("transaction verified")

	// check PoW for Sybil attack prevention
	now := time.Now()
	log.Info("PoW started")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute) // TODO: make this configurable
	checked := svc.vm.sybilControl.Check(ctx, pow.NewUnit(args.Transaction.Bytes()))
	cancel()
	if !checked {
		reply.Error = ErrPoWFailed
		reply.Success = false
		return ErrPoWFailed
	}
	log.Info("PoW completed", "took", time.Since(now))

	// assume previous Put, locally serve reads
	if args.Unsigned.Op == "Range" {
		opts := []storage.OpOption{
			storage.WithPublicKey(&ed25519.PublicKey{PublicKey: []byte(args.Unsigned.PublicKey)}),
		}
		if len(args.Unsigned.RangeEnd) > 0 {
			opts = append(opts, storage.WithRangeEnd(args.Unsigned.RangeEnd))
		}
		reply.RangeResponse, err = svc.vm.s.Range([]byte(args.Unsigned.Key), opts...)
		if err != nil {
			reply.Error = err
			reply.Success = false
		}
		reply.Success = true
		reply.TxID = ""
		return err
	}

	// pass it to consensus and persist on accept
	log.Info("initiating consensus")
	svc.vm.mempool.Push(args.Transaction)
	svc.vm.notifyBlockReady()
	log.Info("initiated consensus")

	// TODO: poll? currently done in client/cli side
	reply.TxID = args.Transaction.ID().String()
	reply.Success = true
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

func (svc *Service) CheckTx(_ *http.Request, args *CheckTxArgs, reply *CheckTxReply) (err error) {
	if args.TxID == "" {
		reply.Error = errors.New("empty transaction ID")
		reply.Confirmed = false
		return nil
	}
	txID, err := ids.FromString(args.TxID)
	if err != nil {
		reply.Error = err
		reply.Confirmed = false
		return nil
	}

	reply.Confirmed = svc.vm.isTxConfirmed(txID)
	return nil
}
