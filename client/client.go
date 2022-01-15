// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package client implements "spacesvm" client SDK.
package client

import (
	"context"
	"errors"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/tdata"
	"github.com/ava-labs/spacesvm/vm"
)

// Client defines spacesvm client operations.
type Client interface {
	// Pings the VM.
	Ping() (bool, error)

	// Returns the VM genesis.
	Genesis() (*chain.Genesis, error)
	// Accepted fetches the ID of the last accepted block.
	Accepted() (ids.ID, error)

	// Returns if a space is already claimed
	Claimed(space string) (bool, error)
	// Returns the corresponding space information.
	Info(space string) (*chain.SpaceInfo, []*chain.KeyValue, error)
	// Balance returns the balance of an account
	Balance(addr common.Address) (bal uint64, err error)
	// Resolve returns the value associated with a path
	Resolve(path string) (exists bool, value []byte, err error)

	// Requests the suggested price and cost from VM.
	SuggestedRawFee() (uint64, uint64, error)
	// Issues the transaction and returns the transaction ID.
	IssueRawTx(d []byte) (ids.ID, error)

	// Requests the suggested price and cost from VM, returns the input as
	// TypedData.
	SuggestedFee(i *chain.Input) (*tdata.TypedData, uint64, error)
	// Issues a human-readable transaction and returns the transaction ID.
	IssueTx(td *tdata.TypedData, sig []byte) (ids.ID, error)

	// Checks the status of the transaction, and returns "true" if confirmed.
	HasTx(id ids.ID) (bool, error)
	// Polls the transactions until its status is confirmed.
	PollTx(ctx context.Context, txID ids.ID) (confirmed bool, err error)
}

// New creates a new client object.
func New(uri string, reqTimeout time.Duration) Client {
	req := rpc.NewEndpointRequester(
		uri,
		vm.PublicEndpoint,
		"spacesvm",
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

func (cli *client) Claimed(space string) (bool, error) {
	resp := new(vm.ClaimedReply)
	if err := cli.req.SendRequest(
		"claimed",
		&vm.ClaimedArgs{Space: space},
		resp,
	); err != nil {
		return false, err
	}
	return resp.Claimed, nil
}

func (cli *client) Info(space string) (*chain.SpaceInfo, []*chain.KeyValue, error) {
	resp := new(vm.InfoReply)
	if err := cli.req.SendRequest(
		"info",
		&vm.InfoArgs{Space: space},
		resp,
	); err != nil {
		return nil, nil, err
	}
	return resp.Info, resp.Values, nil
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

func (cli *client) SuggestedRawFee() (uint64, uint64, error) {
	resp := new(vm.SuggestedRawFeeReply)
	if err := cli.req.SendRequest(
		"suggestedRawFee",
		nil,
		resp,
	); err != nil {
		return 0, 0, err
	}
	return resp.Price, resp.Cost, nil
}

func (cli *client) IssueRawTx(d []byte) (ids.ID, error) {
	resp := new(vm.IssueRawTxReply)
	if err := cli.req.SendRequest(
		"issueRawTx",
		&vm.IssueRawTxArgs{Tx: d},
		resp,
	); err != nil {
		return ids.Empty, err
	}
	return resp.TxID, nil
}

func (cli *client) HasTx(txID ids.ID) (bool, error) {
	resp := new(vm.HasTxReply)
	if err := cli.req.SendRequest(
		"hasTx",
		&vm.HasTxArgs{TxID: txID},
		resp,
	); err != nil {
		return false, err
	}
	return resp.Accepted, nil
}

func (cli *client) SuggestedFee(i *chain.Input) (*tdata.TypedData, uint64, error) {
	resp := new(vm.SuggestedFeeReply)
	if err := cli.req.SendRequest(
		"suggestedFee",
		&vm.SuggestedFeeArgs{Input: i},
		resp,
	); err != nil {
		return nil, 0, err
	}
	return resp.TypedData, resp.TotalCost, nil
}

func (cli *client) IssueTx(td *tdata.TypedData, sig []byte) (ids.ID, error) {
	resp := new(vm.IssueTxReply)
	if err := cli.req.SendRequest(
		"issueTx",
		&vm.IssueTxArgs{TypedData: td, Signature: sig},
		resp,
	); err != nil {
		return ids.Empty, err
	}

	return resp.TxID, nil
}

func (cli *client) PollTx(ctx context.Context, txID ids.ID) (confirmed bool, err error) {
done:
	for ctx.Err() == nil {
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			break done
		}

		confirmed, err := cli.HasTx(txID)
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

func (cli *client) IssueTxHR(d []byte, sig []byte) (ids.ID, error) {
	return ids.ID{}, errors.New("not implemented")
}

func (cli *client) Balance(addr common.Address) (bal uint64, err error) {
	resp := new(vm.BalanceReply)
	if err = cli.req.SendRequest(
		"balance",
		&vm.BalanceArgs{
			Address: addr,
		},
		resp,
	); err != nil {
		return 0, err
	}
	return resp.Balance, nil
}

type Op struct {
	pollTx bool
	space  string
}

type OpOption func(*Op)

func (op *Op) applyOpts(opts []OpOption) {
	for _, opt := range opts {
		opt(op)
	}
}

// "true" to poll transaction for its confirmation.
func WithPollTx() OpOption {
	return func(op *Op) { op.pollTx = true }
}

// Non-empty to print out space information.
func WithInfo(space string) OpOption {
	return func(op *Op) { op.space = space }
}
