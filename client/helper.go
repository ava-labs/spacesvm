// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/tdata"
)

func PPInfo(info *chain.SpaceInfo) {
	expiry := time.Unix(int64(info.Expiry), 0)
	color.Cyan(
		"raw space %s: units=%d expiry=%v (%v remaining)",
		info.RawSpace, info.Units, expiry, time.Until(expiry),
	)
}

func PPActivity(a []*chain.Activity) error {
	if len(a) == 0 {
		color.Cyan("no recent activity")
	}
	for _, item := range a {
		b, err := json.Marshal(item)
		if err != nil {
			return err
		}
		color.Cyan(string(b))
	}
	return nil
}

// Signs and issues the transaction (node construction).
func SignIssueTx(
	ctx context.Context,
	cli Client,
	input *chain.Input,
	priv *ecdsa.PrivateKey,
	opts ...OpOption,
) (txID ids.ID, cost uint64, err error) {
	ret := &Op{}
	ret.applyOpts(opts)

	td, txCost, err := cli.SuggestedFee(ctx, input)
	if err != nil {
		return ids.Empty, 0, err
	}

	dh, err := tdata.DigestHash(td)
	if err != nil {
		return ids.Empty, 0, fmt.Errorf("%w: failed to compute digest hash", err)
	}

	sig, err := chain.Sign(dh, priv)
	if err != nil {
		return ids.Empty, 0, err
	}

	txID, err = cli.IssueTx(ctx, td, sig)
	if err != nil {
		return ids.Empty, 0, err
	}

	if err := handleConfirmation(ctx, ret, cli, txID, priv); err != nil {
		return ids.Empty, 0, err
	}
	return txID, txCost, nil
}

// Signs and issues the transaction (local construction).
func SignIssueRawTx(
	ctx context.Context,
	cli Client,
	utx chain.UnsignedTransaction,
	priv *ecdsa.PrivateKey,
	opts ...OpOption,
) (txID ids.ID, cost uint64, err error) {
	ret := &Op{}
	ret.applyOpts(opts)

	g, err := cli.Genesis(ctx)
	if err != nil {
		return ids.Empty, 0, err
	}

	la, err := cli.Accepted(ctx)
	if err != nil {
		return ids.Empty, 0, err
	}

	price, blockCost, err := cli.SuggestedRawFee(ctx)
	if err != nil {
		return ids.Empty, 0, err
	}

	utx.SetBlockID(la)
	utx.SetMagic(g.Magic)
	utx.SetPrice(price + blockCost/utx.FeeUnits(g))

	dh, err := chain.DigestHash(utx)
	if err != nil {
		return ids.Empty, 0, err
	}

	sig, err := chain.Sign(dh, priv)
	if err != nil {
		return ids.Empty, 0, err
	}

	tx := chain.NewTx(utx, sig)
	if err := tx.Init(g); err != nil {
		return ids.Empty, 0, err
	}

	color.Yellow(
		"issuing tx %s (fee units=%d, load units=%d, price=%d, blkID=%s)",
		tx.ID(), tx.FeeUnits(g), tx.LoadUnits(g), tx.GetPrice(), tx.GetBlockID(),
	)
	txID, err = cli.IssueRawTx(ctx, tx.Bytes())
	if err != nil {
		return ids.Empty, 0, err
	}

	if err := handleConfirmation(ctx, ret, cli, txID, priv); err != nil {
		return ids.Empty, 0, err
	}
	return txID, utx.GetPrice() * utx.FeeUnits(g), nil
}

func handleConfirmation(
	ctx context.Context, ret *Op, cli Client,
	txID ids.ID, priv *ecdsa.PrivateKey,
) error {
	if ret.pollTx {
		color.Yellow("issued transaction %s (now polling)", txID)
		confirmed, err := cli.PollTx(ctx, txID)
		if err != nil {
			return err
		}
		if !confirmed {
			color.Yellow("transaction %s not confirmed", txID)
		} else {
			color.Yellow("transaction %s confirmed", txID)
		}
	}

	if len(ret.space) > 0 {
		info, _, err := cli.Info(ctx, ret.space)
		if err != nil {
			color.Red("cannot get space info %v", err)
			return err
		}
		PPInfo(info)
	}

	if ret.balance {
		addr := crypto.PubkeyToAddress(priv.PublicKey)
		b, err := cli.Balance(ctx, addr)
		if err != nil {
			return err
		}
		color.Cyan("Address=%s Balance=%d", addr, b)
	}

	return nil
}

type Op struct {
	pollTx  bool
	space   string
	balance bool
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

func WithBalance() OpOption {
	return func(op *Op) { op.balance = true }
}
