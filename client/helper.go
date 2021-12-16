// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import (
	"context"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/crypto"
	"github.com/fatih/color"
)

// Mines against the unsigned transaction first.
// And signs and issues the transaction.
func MineSignIssueTx(
	cli Client,
	timeout time.Duration,
	utx chain.UnsignedTransaction,
	priv *crypto.PrivateKey,
) (txID ids.ID, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	mtx, err := cli.Mine(ctx, utx)
	cancel()
	if err != nil {
		return ids.Empty, err
	}

	b, err := chain.UnsignedBytes(mtx)
	if err != nil {
		return ids.Empty, err
	}
	sig, err := priv.Sign(b)
	if err != nil {
		return ids.Empty, err
	}

	tx := chain.NewTx(mtx, sig)
	if err := tx.Init(); err != nil {
		return ids.Empty, err
	}

	color.Yellow("issuing tx %s with block ID %s", tx.ID(), mtx.GetBlockID())
	txID, err = cli.IssueTx(tx.Bytes())
	if err != nil {
		return ids.Empty, err
	}

	return txID, nil
}
