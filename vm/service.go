// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/parser"
)

var (
	ErrInvalidEmptyTx = errors.New("invalid empty transaction")
)

type PublicService struct {
	vm *VM
}

type PingReply struct {
	Success bool `serialize:"true" json:"success"`
}

func (svc *PublicService) Ping(_ *http.Request, _ *struct{}, reply *PingReply) (err error) {
	log.Info("ping")
	reply.Success = true
	return nil
}

type GenesisReply struct {
	Genesis *chain.Genesis `serialize:"true" json:"genesis"`
}

func (svc *PublicService) Genesis(_ *http.Request, _ *struct{}, reply *GenesisReply) (err error) {
	reply.Genesis = svc.vm.Genesis()
	return nil
}

type IssueTxArgs struct {
	Tx []byte `serialize:"true" json:"tx"`
}

type IssueTxReply struct {
	TxID    ids.ID `serialize:"true" json:"txId"`
	Success bool   `serialize:"true" json:"success"`
}

func (svc *PublicService) IssueTx(_ *http.Request, args *IssueTxArgs, reply *IssueTxReply) error {
	tx := new(chain.Transaction)
	if _, err := chain.Unmarshal(args.Tx, tx); err != nil {
		return err
	}

	// otherwise, unexported tx.id field is empty
	if err := tx.Init(svc.vm.genesis); err != nil {
		reply.Success = false
		return err
	}
	reply.TxID = tx.ID()

	errs := svc.vm.Submit(tx)
	reply.Success = len(errs) == 0
	if reply.Success {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return fmt.Errorf("%v", errs)
}

type CheckTxArgs struct {
	TxID ids.ID `serialize:"true" json:"txId"`
}

type CheckTxReply struct {
	Confirmed bool `serialize:"true" json:"confirmed"`
}

func (svc *PublicService) CheckTx(_ *http.Request, args *CheckTxArgs, reply *CheckTxReply) error {
	has, err := chain.HasTransaction(svc.vm.db, args.TxID)
	if err != nil {
		return err
	}
	reply.Confirmed = has
	return nil
}

type LastAcceptedReply struct {
	BlockID ids.ID `serialize:"true" json:"blockId"`
}

func (svc *PublicService) LastAccepted(_ *http.Request, _ *struct{}, reply *LastAcceptedReply) error {
	reply.BlockID = svc.vm.lastAccepted.ID()
	return nil
}

type ValidBlockIDArgs struct {
	BlockID ids.ID `serialize:"true" json:"blockId"`
}

type ValidBlockIDReply struct {
	Valid bool `serialize:"true" json:"valid"`
}

type SuggestedFeeArgs struct{}

type SuggestedFeeReply struct {
	Price uint64 `serialize:"true" json:"price"`
	Cost  uint64 `serialize:"true" json:"cost"`
}

func (svc *PublicService) SuggestedFee(
	_ *http.Request,
	_ *SuggestedFeeArgs,
	reply *SuggestedFeeReply,
) error {
	price, cost, err := svc.vm.SuggestedFee()
	if err != nil {
		return err
	}
	reply.Price = price
	reply.Cost = cost
	return nil
}

type ClaimedArgs struct {
	Space string `serialize:"true" json:"space"`
}

type ClaimedReply struct {
	Claimed bool `serialize:"true" json:"claimed"`
}

func (svc *PublicService) Claimed(_ *http.Request, args *ClaimedArgs, reply *ClaimedReply) error {
	if err := parser.CheckContents(args.Space); err != nil {
		return err
	}
	has, err := chain.HasSpace(svc.vm.db, []byte(args.Space))
	if err != nil {
		return err
	}
	reply.Claimed = has
	return nil
}

type InfoArgs struct {
	Space string `serialize:"true" json:"space"`
}

type InfoReply struct {
	Info   *chain.SpaceInfo  `serialize:"true" json:"info"`
	Values []*chain.KeyValue `serialize:"true" json:"pairs"`
}

func (svc *PublicService) Info(_ *http.Request, args *InfoArgs, reply *InfoReply) error {
	if err := parser.CheckContents(args.Space); err != nil {
		return err
	}

	i, _, err := chain.GetSpaceInfo(svc.vm.db, []byte(args.Space))
	if err != nil {
		return err
	}

	kvs, err := chain.GetAllValues(svc.vm.db, i.RawSpace)
	if err != nil {
		return err
	}
	reply.Info = i
	reply.Values = kvs
	return nil
}

type ResolveArgs struct {
	Path string `serialize:"true" json:"path"`
}

type ResolveReply struct {
	Exists bool   `serialize:"true" json:"exists"`
	Value  []byte `serialize:"true" json:"value"`
}

func (svc *PublicService) Resolve(_ *http.Request, args *ResolveArgs, reply *ResolveReply) error {
	space, key, err := parser.ResolvePath(args.Path)
	if err != nil {
		return err
	}

	v, exists, err := chain.GetValue(svc.vm.db, []byte(space), []byte(key))
	reply.Exists = exists
	reply.Value = v
	return err
}

type BalanceArgs struct {
	Address string `serialize:"true" json:"address"`
}

type BalanceReply struct {
	Balance uint64 `serialize:"true" json:"balance"`
}

func (svc *PublicService) Balance(_ *http.Request, args *BalanceArgs, reply *BalanceReply) error {
	paddr := common.HexToAddress(args.Address)
	bal, err := chain.GetBalance(svc.vm.db, paddr)
	if err != nil {
		return err
	}
	reply.Balance = bal
	return err
}

// TODO: SuggestFeeHR, IssueTxHR
