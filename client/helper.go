// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import (
	"context"
	"errors"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/fatih/color"

	"github.com/ava-labs/quarkvm/chain"
)

// Mines against the unsigned transaction first.
// And signs and issues the transaction.
func MineSignIssueTx(
	ctx context.Context,
	cli Client,
	rtx chain.UnsignedTransaction,
	priv crypto.PrivateKey,
	opts ...OpOption,
) (txID ids.ID, err error) {
	ret := &Op{}
	ret.applyOpts(opts)

	g, err := cli.Genesis()
	if err != nil {
		return ids.Empty, err
	}

	utx, err := cli.Mine(ctx, g, rtx)
	if err != nil {
		return ids.Empty, err
	}

	b, err := chain.UnsignedBytes(utx)
	if err != nil {
		return ids.Empty, err
	}
	parsedPriv, ok := priv.(*crypto.PrivateKeySECP256K1R)
	if !ok {
		return ids.Empty, errors.New("incorrect key type")
	}
	sig, err := parsedPriv.Sign(b)
	if err != nil {
		return ids.Empty, err
	}

	tx := chain.NewTx(utx, sig)
	if err := tx.Init(g); err != nil {
		return ids.Empty, err
	}

	color.Yellow(
		"issuing tx %s (fee units=%d, load units=%d, difficulty=%d, blkID=%s)",
		tx.ID(), tx.FeeUnits(g), tx.LoadUnits(g), tx.Difficulty(), tx.GetBlockID(),
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

	if len(ret.prefixInfo) > 0 {
		info, err := cli.PrefixInfo(ret.prefixInfo)
		if err != nil {
			color.Red("cannot get prefix info %v", err)
			return ids.Empty, err
		}
		expiry := time.Unix(int64(info.Expiry), 0)
		color.Blue(
			"raw prefix %s: units=%d expiry=%v (%v remaining)",
			info.RawPrefix, info.Units, expiry, time.Until(expiry),
		)
	}

	return txID, nil
}
