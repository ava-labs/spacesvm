// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
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

type NetworkReply struct {
	NetworkID uint32 `serialize:"true" json:"networkId"`
	SubnetID  ids.ID `serialize:"true" json:"subnetId"`
	ChainID   ids.ID `serialize:"true" json:"chainId"`
}

func (svc *PublicService) Network(_ *http.Request, _ *struct{}, reply *NetworkReply) (err error) {
	reply.NetworkID = svc.vm.ctx.NetworkID
	reply.SubnetID = svc.vm.ctx.SubnetID
	reply.ChainID = svc.vm.ctx.ChainID
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
	TxID ids.ID `serialize:"true" json:"txId"`
}

func (svc *PublicService) IssueRawTx(_ *http.Request, args *IssueRawTxArgs, reply *IssueRawTxReply) error {
	tx := new(chain.Transaction)
	if _, err := chain.Unmarshal(args.Tx, tx); err != nil {
		return err
	}

	// otherwise, unexported tx.id field is empty
	if err := tx.Init(svc.vm.genesis); err != nil {
		return err
	}
	reply.TxID = tx.ID()

	errs := svc.vm.Submit(tx)
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return fmt.Errorf("%v", errs)
}

type IssueTxArgs struct {
	TypedData *tdata.TypedData `serialize:"true" json:"typedData"`
	Signature hexutil.Bytes    `serialize:"true" json:"signature"`
}

type IssueTxReply struct {
	TxID ids.ID `serialize:"true" json:"txId"`
}

func (svc *PublicService) IssueTx(_ *http.Request, args *IssueTxArgs, reply *IssueTxReply) error {
	if args.TypedData == nil {
		return ErrTypedDataIsNil
	}
	utx, err := chain.ParseTypedData(args.TypedData)
	if err != nil {
		return err
	}
	tx := chain.NewTx(utx, args.Signature[:])

	// otherwise, unexported tx.id field is empty
	if err := tx.Init(svc.vm.genesis); err != nil {
		return err
	}
	reply.TxID = tx.ID()

	errs := svc.vm.Submit(tx)
	if len(errs) == 0 {
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
	Accepted bool `serialize:"true" json:"accepted"`
}

func (svc *PublicService) HasTx(_ *http.Request, args *HasTxArgs, reply *HasTxReply) error {
	has, err := chain.HasTransaction(svc.vm.db, args.TxID)
	if err != nil {
		return err
	}
	reply.Accepted = has
	return nil
}

type LastAcceptedReply struct {
	Height  uint64 `serialize:"true" json:"height"`
	BlockID ids.ID `serialize:"true" json:"blockId"`
}

func (svc *PublicService) LastAccepted(_ *http.Request, _ *struct{}, reply *LastAcceptedReply) error {
	la := svc.vm.lastAccepted
	reply.Height = la.Hght
	reply.BlockID = la.ID()
	return nil
}

type SuggestedFeeArgs struct {
	Input *chain.Input `serialize:"true" json:"input"`
}

type SuggestedFeeReply struct {
	TypedData *tdata.TypedData `serialize:"true" json:"typedData"`
	TotalCost uint64           `serialize:"true" json:"totalCost"`
}

func (svc *PublicService) SuggestedFee(
	_ *http.Request,
	args *SuggestedFeeArgs,
	reply *SuggestedFeeReply,
) error {
	if args.Input == nil {
		return ErrInputIsNil
	}
	utx, err := args.Input.Decode()
	if err != nil {
		return err
	}

	// Determine suggested fee
	price, cost, err := svc.vm.SuggestedFee()
	if err != nil {
		return err
	}
	g := svc.vm.genesis
	fu := utx.FeeUnits(g)
	price += cost / fu

	// Update meta
	utx.SetBlockID(svc.vm.lastAccepted.ID())
	utx.SetMagic(g.Magic)
	utx.SetPrice(price)

	reply.TypedData = utx.TypedData()
	reply.TotalCost = fu * price
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
	Info   *chain.SpaceInfo      `serialize:"true" json:"info"`
	Values []*chain.KeyValueMeta `serialize:"true" json:"values"`
}

func (svc *PublicService) Info(_ *http.Request, args *InfoArgs, reply *InfoReply) error {
	if err := parser.CheckContents(args.Space); err != nil {
		return err
	}

	i, exists, err := chain.GetSpaceInfo(svc.vm.db, []byte(args.Space))
	if err != nil {
		return err
	}
	if !exists {
		return chain.ErrSpaceMissing
	}

	kvs, err := chain.GetAllValueMetas(svc.vm.db, i.RawSpace)
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
	Exists    bool             `serialize:"true" json:"exists"`
	Value     []byte           `serialize:"true" json:"value"`
	ValueMeta *chain.ValueMeta `serialize:"true" json:"valueMeta"`
}

func (svc *PublicService) Resolve(_ *http.Request, args *ResolveArgs, reply *ResolveReply) error {
	space, key, err := parser.ResolvePath(args.Path)
	if err != nil {
		return err
	}

	vmeta, exists, err := chain.GetValueMeta(svc.vm.db, []byte(space), []byte(key))
	if err != nil {
		return err
	}
	if !exists {
		// Avoid value lookup if doesn't exist
		return nil
	}
	v, exists, err := chain.GetValue(svc.vm.db, []byte(space), []byte(key))
	if err != nil {
		return err
	}
	if !exists {
		return ErrCorruption
	}

	// Set values properly
	reply.Exists = true
	reply.Value = v
	reply.ValueMeta = vmeta
	return nil
}

type BalanceArgs struct {
	Address common.Address `serialize:"true" json:"address"`
}

type BalanceReply struct {
	Balance uint64 `serialize:"true" json:"balance"`
}

func (svc *PublicService) Balance(_ *http.Request, args *BalanceArgs, reply *BalanceReply) error {
	bal, err := chain.GetBalance(svc.vm.db, args.Address)
	if err != nil {
		return err
	}
	reply.Balance = bal
	return err
}

type RecentActivityReply struct {
	Activity []*chain.Activity `serialize:"true" json:"activity"`
}

func (svc *PublicService) RecentActivity(_ *http.Request, _ *struct{}, reply *RecentActivityReply) error {
	cs := uint64(svc.vm.config.ActivityCacheSize)
	if cs == 0 {
		return nil
	}

	// Sort results from newest to oldest
	start := svc.vm.activityCacheCursor
	i := start
	activity := []*chain.Activity{}
	for i > 0 && start-i < cs {
		i--
		item := svc.vm.activityCache[i%cs]
		if item == nil {
			break
		}
		activity = append(activity, item)
	}
	reply.Activity = activity
	return nil
}

type OwnedArgs struct {
	Address common.Address `serialize:"true" json:"address"`
}

type OwnedReply struct {
	Spaces []string `serialize:"true" json:"spaces"`
}

func (svc *PublicService) Owned(_ *http.Request, args *OwnedArgs, reply *OwnedReply) error {
	spaces, err := chain.GetAllOwned(svc.vm.db, args.Address)
	if err != nil {
		return err
	}
	reply.Spaces = spaces
	return nil
}
