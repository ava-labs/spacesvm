// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package client implements "quarkvm" client SDK.
package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/parser"
	"github.com/ava-labs/quarkvm/vm"
	"github.com/fatih/color"
)

// Client defines quarkvm client operations.
type Client interface {
	// Pings the VM.
	Ping() (bool, error)
	// Returns the corresponding prefix information.
	PrefixInfo(pfx []byte) (*chain.PrefixInfo, error)
	// Preferred fetches the ID of the currently preferred block.
	Preferred() (ids.ID, error)
	// Checks the validity of the block.
	// Returns "true" if the block is valid.
	CheckBlock(blkID ids.ID) (bool, error)
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
	// Performs Proof-of-Work (PoW) by enumerating the graffiti.
	Mine(
		ctx context.Context, utx chain.UnsignedTransaction, difficulty uint64, minSurplus uint64,
	) (chain.UnsignedTransaction, error)
}

// New creates a new client object.
func New(uri string, endpoint string, reqTimeout time.Duration) Client {
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	req := rpc.NewEndpointRequester(
		uri,
		endpoint,
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
		&vm.PingArgs{},
		resp,
	)
	if err != nil {
		return false, err
	}
	return resp.Success, nil
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

func (cli *client) Preferred() (ids.ID, error) {
	resp := new(vm.CurrBlockReply)
	if err := cli.req.SendRequest(
		"currBlock",
		&vm.CurrBlockArgs{},
		resp,
	); err != nil {
		color.Red("failed to get curr block %v", err)
		return ids.ID{}, err
	}
	return resp.BlockID, nil
}

func (cli *client) CheckBlock(blkID ids.ID) (bool, error) {
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

func (cli *client) Mine(
	ctx context.Context, utx chain.UnsignedTransaction, difficulty uint64, minSurplus uint64,
) (chain.UnsignedTransaction, error) {
	for ctx.Err() == nil {
		// TODO: only query this periodically
		cbID, err := cli.Preferred()
		if err != nil {
			return nil, err
		}
		utx.SetBlockID(cbID)

		graffiti := uint64(0)
		for ctx.Err() == nil {
			// TODO: only query periodically
			valid, err := cli.CheckBlock(cbID)
			if err != nil {
				return nil, err
			}
			if !valid {
				color.Yellow("%v is no longer a valid block id", cbID)
				break
			}
			utx.SetGraffiti(graffiti)
			_, utxd, err := chain.CalcDifficulty(utx)
			if err != nil {
				return nil, err
			}
			if utxd >= difficulty && (utxd-difficulty)*utx.Units() >= minSurplus {
				return utx, nil
			}
			graffiti++
		}
		// TODO: get new block hash if no longer valid
	}
	return nil, ctx.Err()
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
