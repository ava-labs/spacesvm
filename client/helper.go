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
	color.Blue(
		"raw prefix %s: units=%d expiry=%v (%v remaining)",
		info.RawSpace, info.Units, expiry, time.Until(expiry),
	)
}

func PPActivity(a []*chain.Activity) error {
	if len(a) == 0 {
		color.Blue("no recent activity")
	}
	for _, item := range a {
		b, err := json.Marshal(item)
		if err != nil {
			return err
		}
		color.Blue(string(b))
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
) (txID ids.ID, err error) {
	ret := &Op{}
	ret.applyOpts(opts)

	td, cost, err := cli.SuggestedFee(input)
	if err != nil {
		return ids.Empty, err
	}
	// Log typed data
	b, err := json.Marshal(td)
	if err != nil {
		return ids.Empty, err
	}
	color.Cyan("typed data (cost=%d): %v", cost, string(b))

	dh, err := tdata.DigestHash(td)
	if err != nil {
		return ids.Empty, fmt.Errorf("%w: failed to compute digest hash", err)
	}

	sig, err := crypto.Sign(dh, priv)
	if err != nil {
		return ids.Empty, err
	}

	txID, err = cli.IssueTx(td, sig)
	if err != nil {
		return ids.Empty, err
	}

	if ret.pollTx {
		color.Green("issued transaction %s (now polling)", txID)
		confirmed, err := cli.PollTx(ctx, txID)
		if err != nil {
			return ids.Empty, err
		}
		if !confirmed {
			color.Yellow("transaction %s not confirmed", txID)
		} else {
			color.Green("transaction %s confirmed", txID)
		}
	}

	if len(ret.space) > 0 {
		info, _, err := cli.Info(ret.space)
		if err != nil {
			color.Red("cannot get prefix info %v", err)
			return ids.Empty, err
		}
		PPInfo(info)
	}

	return txID, nil
}

// Signs and issues the transaction (local construction).
func SignIssueRawTx(
	ctx context.Context,
	cli Client,
	utx chain.UnsignedTransaction,
	priv *ecdsa.PrivateKey,
	opts ...OpOption,
) (txID ids.ID, err error) {
	ret := &Op{}
	ret.applyOpts(opts)

	g, err := cli.Genesis()
	if err != nil {
		return ids.Empty, err
	}

	la, err := cli.Accepted()
	if err != nil {
		return ids.Empty, err
	}

	price, blockCost, err := cli.SuggestedRawFee()
	if err != nil {
		return ids.Empty, err
	}

	utx.SetBlockID(la)
	utx.SetMagic(g.Magic)
	utx.SetPrice(price + blockCost/utx.FeeUnits(g))

	// Log typed data
	b, err := json.Marshal(utx.TypedData())
	if err != nil {
		return ids.Empty, err
	}
	color.Cyan("typed data: %v", string(b))

	dh, err := chain.DigestHash(utx)
	if err != nil {
		return ids.Empty, err
	}

	sig, err := crypto.Sign(dh, priv)
	if err != nil {
		return ids.Empty, err
	}

	tx := chain.NewTx(utx, sig)
	if err := tx.Init(g); err != nil {
		return ids.Empty, err
	}

	color.Yellow(
		"issuing tx %s (fee units=%d, load units=%d, price=%d, blkID=%s)",
		tx.ID(), tx.FeeUnits(g), tx.LoadUnits(g), tx.GetPrice(), tx.GetBlockID(),
	)
	txID, err = cli.IssueRawTx(tx.Bytes())
	if err != nil {
		return ids.Empty, err
	}

	if ret.pollTx {
		color.Green("issued transaction %s (now polling)", txID)
		confirmed, err := cli.PollTx(ctx, txID)
		if err != nil {
			return ids.Empty, err
		}
		if !confirmed {
			color.Yellow("transaction %s not confirmed", txID)
		} else {
			color.Green("transaction %s confirmed", txID)
		}
	}

	if len(ret.space) > 0 {
		info, _, err := cli.Info(ret.space)
		if err != nil {
			color.Red("cannot get prefix info %v", err)
			return ids.Empty, err
		}
		PPInfo(info)
	}

	return txID, nil
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
