// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/parser"
	"github.com/ava-labs/spacesvm/tdata"
)

var ErrInvalidEmptyTx = errors.New("invalid empty transaction")

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

type IssueRawTxArgs struct {
	Tx []byte `serialize:"true" json:"tx"`
}

type IssueRawTxReply struct {
	TxID    ids.ID `serialize:"true" json:"txId"`
	Success bool   `serialize:"true" json:"success"`
}

func (svc *PublicService) IssueRawTx(_ *http.Request, args *IssueRawTxArgs, reply *IssueRawTxReply) error {
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

type IssueTxArgs struct {
	TypedData tdata.TypedData           `serialize:"true" json:"typedData"`
	Utx       chain.UnsignedTransaction `serialize:"true" json:"unsignedTx"`
	Signature hexutil.Bytes             `serialize:"true" json:"signature"`
}

type IssueTxReply struct {
	TxID    ids.ID `serialize:"true" json:"txId"`
	Success bool   `serialize:"true" json:"success"`
}

func (svc *PublicService) IssueTx(_ *http.Request, args *IssueTxArgs, reply *IssueTxReply) error {
	// TODO: ensure both are valid
	tx := chain.NewTx(args.Utx, args.Signature[:])

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

type HasTxArgs struct {
	TxID ids.ID `serialize:"true" json:"txId"`
}

type HasTxReply struct {
	Confirmed bool `serialize:"true" json:"confirmed"`
}

func (svc *PublicService) HasTx(_ *http.Request, args *HasTxArgs, reply *HasTxReply) error {
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

type SuggestedFeeArgs struct {
	Input *chain.Input `serialize:"true" json:"input"`
}

type SuggestedFeeReply struct {
	TypedData *tdata.TypedData          `serialize:"true" json:"typedData"`
	Utx       chain.UnsignedTransaction `serialize:"true" json:"unsignedTx"`
	TotalCost uint64                    `serialize:"true" json:"totalCost"`
}

func (svc *PublicService) SuggestedFee(
	_ *http.Request,
	args *SuggestedFeeArgs,
	reply *SuggestedFeeReply,
) error {
	if args.Input == nil {
		return errors.New("input is empty")
	}
	utx, err := args.Input.Decode()
	if err != nil {
		return err
	}

	g := svc.vm.genesis
	price, cost, err := svc.vm.SuggestedFee()
	if err != nil {
		return err
	}
	fu := utx.FeeUnits(g)
	price += cost / fu
	utx.SetBlockID(svc.vm.lastAccepted.ID())
	utx.SetMagic(g.Magic)
	utx.SetPrice(price)

	reply.TypedData = utx.TypedData()
	reply.TotalCost = fu * price
	reply.Utx = utx
	return nil
}

type SuggestedRawFeeReply struct {
	Price uint64 `serialize:"true" json:"price"`
	Cost  uint64 `serialize:"true" json:"cost"`
}

func (svc *PublicService) SuggestedRawFee(
	_ *http.Request,
	_ *struct{},
	reply *SuggestedRawFeeReply,
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
