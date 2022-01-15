// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"os"
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

// TODO: move all this to a separate client code
func transferFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	to, units := getTransferOp(args)
	cli := client.New(uri, requestTimeout)

	utx := &chain.TransferTx{
		BaseTx: &chain.BaseTx{},
		To:     to,
		Units:  units,
	}

	opts := []client.OpOption{client.WithPollTx()}
	_, err = client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...)
	if err != nil {
		return err
	}

	addr := crypto.PubkeyToAddress(priv.PublicKey)
	b, err := cli.Balance(addr)
	if err != nil {
		return err
	}
	color.Cyan("Address=%s Balance=%d", addr, b)
	return nil
}

func getTransferOp(args []string) (to common.Address, units uint64) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "expected exactly 2 arguments, got %d", len(args))
		os.Exit(128)
	}

	addr := common.HexToAddress(args[0])
	units, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse units %v", err)
		os.Exit(128)
	}
	return addr, units
}
