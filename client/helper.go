// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import (
	"context"
	"crypto/ecdsa"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"

	"github.com/ava-labs/spacesvm/chain"
)

// Signs and issues the transaction.
func SignIssueTx(
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

	price, blockCost, err := cli.SuggestedFee()
	if err != nil {
		return ids.Empty, err
	}

	utx.SetBlockID(la)
	utx.SetMagic(g.Magic)
	utx.SetPrice(price + blockCost/utx.FeeUnits(g))

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
	txID, err = cli.IssueTx(tx.Bytes())
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
		expiry := time.Unix(int64(info.Expiry), 0)
		color.Blue(
			"raw prefix %s: units=%d expiry=%v (%v remaining)",
			info.RawSpace, info.Units, expiry, time.Until(expiry),
		)
	}

	return txID, nil
}
