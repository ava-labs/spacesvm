// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
)

var transferCmd = &cobra.Command{
	Use:   "transfer [options] <to> <units>",
	Short: "Transfers units to another address",
	RunE:  transferFunc,
}

func transferFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	to, units, err := getTransferOp(args)
	if err != nil {
		return err
	}

	utx := &chain.TransferTx{
		BaseTx: &chain.BaseTx{},
		To:     to,
		Units:  units,
	}

	cli := client.New(uri, requestTimeout)
	opts := []client.OpOption{client.WithPollTx()}
	if verbose {
		opts = append(opts, client.WithBalance())
	}
	if _, _, err := client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...); err != nil {
		return err
	}

	color.Green("transferred %d to %s", units, to.Hex())
	return nil
}

func getTransferOp(args []string) (to common.Address, units uint64, err error) {
	if len(args) != 2 {
		return common.Address{}, 0, fmt.Errorf("expected exactly 2 arguments, got %d", len(args))
	}

	addr := common.HexToAddress(args[0])
	units, err = strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return common.Address{}, 0, fmt.Errorf("%w: failed to parse units", err)
	}
	return addr, units, nil
}
