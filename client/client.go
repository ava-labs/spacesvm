// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package client implements "quarkvm" client SDK.
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/fatih/color"

	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/parser"
	"github.com/ava-labs/quarkvm/vm"
)

// Client defines quarkvm client operations.
type Client interface {
	// Pings the VM.
	Ping() (bool, error)
	// Returns the VM genesis.
	Genesis() (*chain.Genesis, error)
	// Returns if a prefix is already claimed
	Claimed(pfx []byte) (bool, error)
	// Returns the corresponding prefix information.
	PrefixInfo(pfx []byte) (*chain.PrefixInfo, error)
	// Accepted fetches the ID of the last accepted block.
	Accepted() (ids.ID, error)
	// Checks the validity of the blockID.
	// Returns "true" if the block is valid.
	ValidBlockID(blkID ids.ID) (bool, error)
	// Requests for the estimated difficulty from VM.
	EstimateDifficulty() (uint64, uint64, error)
	// Issues the transaction and returns the transaction ID.
	IssueTx(d []byte) (ids.ID, error)
	// Checks the status of the transaction, and returns "true" if confirmed.
	CheckTx(id ids.ID) (bool, error)
	// Polls the transactions until its status is confirmed.
	PollTx(ctx context.Context, txID ids.ID) (confirmed bool, err error)
	// Range runs range-query and returns the results.
	Range(pfx, key []byte, opts ...OpOption) (kvs []chain.KeyValue, err error)
	// Resolve returns the value associated with a path
	Resolve(path string) (exists bool, value []byte, err error)
	// Performs Proof-of-Work (PoW) by enumerating the graffiti.
	Mine(ctx context.Context, gen *chain.Genesis, utx chain.UnsignedTransaction) (chain.UnsignedTransaction, error)
}

// New creates a new client object.
func New(uri string, reqTimeout time.Duration) Client {
	req := rpc.NewEndpointRequester(
		uri,
		vm.PublicEndpoint,
		"quarkvm",
		reqTimeout,
	)
	return &client{req: req}
}

type client struct {
	req rpc.EndpointRequester
}

func (cli *client) Ping() (bool, error) {
	resp := new(vm.PingReply)
	err := cli.req.SendRequest(
		"ping",
		nil,
		resp,
	)
	if err != nil {
		return false, err
	}
	return resp.Success, nil
}

func (cli *client) Genesis() (*chain.Genesis, error) {
	resp := new(vm.GenesisReply)
	err := cli.req.SendRequest(
		"genesis",
		nil,
		resp,
	)
	return resp.Genesis, err
}

func (cli *client) Claimed(pfx []byte) (bool, error) {
	resp := new(vm.ClaimedReply)
	if err := cli.req.SendRequest(
		"claimed",
		&vm.ClaimedArgs{Prefix: pfx},
		resp,
	); err != nil {
		return false, err
	}
	return resp.Claimed, nil
}

func (cli *client) PrefixInfo(pfx []byte) (*chain.PrefixInfo, error) {
	resp := new(vm.PrefixInfoReply)
	if err := cli.req.SendRequest(
		"prefixInfo",
		&vm.PrefixInfoArgs{Prefix: pfx},
		resp,
	); err != nil {
		return nil, err
	}
	return resp.Info, nil
}

func (cli *client) Accepted() (ids.ID, error) {
	resp := new(vm.LastAcceptedReply)
	if err := cli.req.SendRequest(
		"lastAccepted",
		nil,
		resp,
	); err != nil {
		color.Red("failed to get curr block %v", err)
		return ids.ID{}, err
	}
	return resp.BlockID, nil
}

func (cli *client) ValidBlockID(blkID ids.ID) (bool, error) {
	resp := new(vm.ValidBlockIDReply)
	if err := cli.req.SendRequest(
		"validBlockID",
		&vm.ValidBlockIDArgs{BlockID: blkID},
		resp,
	); err != nil {
		return false, err
	}
	return resp.Valid, nil
}

func (cli *client) EstimateDifficulty() (uint64, uint64, error) {
	resp := new(vm.DifficultyEstimateReply)
	if err := cli.req.SendRequest(
		"difficultyEstimate",
		&vm.DifficultyEstimateArgs{},
		resp,
	); err != nil {
		return 0, 0, err
	}
	return resp.Difficulty, resp.Cost, nil
}

func (cli *client) IssueTx(d []byte) (ids.ID, error) {
	resp := new(vm.IssueTxReply)
	if err := cli.req.SendRequest(
		"issueTx",
		&vm.IssueTxArgs{Tx: d},
		resp,
	); err != nil {
		return ids.Empty, err
	}

	txID := resp.TxID
	if !resp.Success {
		return ids.Empty, fmt.Errorf("issue tx %s failed", txID)
	}
	return txID, nil
}

func (cli *client) CheckTx(txID ids.ID) (bool, error) {
	resp := new(vm.CheckTxReply)
	if err := cli.req.SendRequest(
		"checkTx",
		&vm.CheckTxArgs{TxID: txID},
		resp,
	); err != nil {
		return false, err
	}
	return resp.Confirmed, nil
}

func (cli *client) Range(pfx, key []byte, opts ...OpOption) (kvs []chain.KeyValue, err error) {
	ret := &Op{key: key}
	ret.applyOpts(opts)

	resp := new(vm.RangeReply)
	if err = cli.req.SendRequest(
		"range",
		&vm.RangeArgs{
			Prefix:   pfx,
			Key:      key,
			RangeEnd: ret.rangeEnd,
			Limit:    ret.rangeLimit,
		},
		resp,
	); err != nil {
		return nil, err
	}
	kvs = resp.KeyValues
	return kvs, nil
}

func (cli *client) PollTx(ctx context.Context, txID ids.ID) (confirmed bool, err error) {
done:
	for ctx.Err() == nil {
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			break done
		}

		confirmed, err := cli.CheckTx(txID)
		if err != nil {
			color.Red("polling transaction failed %v", err)
			continue
		}
		if confirmed {
			return true, nil
		}
	}
	return false, ctx.Err()
}

func (cli *client) Resolve(path string) (exists bool, value []byte, err error) {
	resp := new(vm.ResolveReply)
	if err = cli.req.SendRequest(
		"resolve",
		&vm.ResolveArgs{
			Path: path,
		},
		resp,
	); err != nil {
		return false, nil, err
	}
	return resp.Exists, resp.Value, nil
}

type Op struct {
	key        []byte
	rangeEnd   []byte
	rangeLimit uint32

	pollTx     bool
	prefixInfo []byte
}

type OpOption func(*Op)

func (op *Op) applyOpts(opts []OpOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func WithPrefix() OpOption {
	return func(op *Op) {
		op.rangeEnd = parser.GetRangeEnd(op.key)
	}
}

// Queries range [start,end).
func WithRangeEnd(end []byte) OpOption {
	return func(op *Op) { op.rangeEnd = end }
}

func WithRangeLimit(limit uint32) OpOption {
	return func(op *Op) { op.rangeLimit = limit }
}

// "true" to poll transaction for its confirmation.
func WithPollTx() OpOption {
	return func(op *Op) { op.pollTx = true }
}

// Non-empty to print out prefix information.
func WithPrefixInfo(pfx []byte) OpOption {
	return func(op *Op) { op.prefixInfo = pfx }
}
