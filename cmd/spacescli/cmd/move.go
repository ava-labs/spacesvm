// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/parser"
)

var moveCmd = &cobra.Command{
	Use:   "move [options] <to> <space>",
	Short: "Transfers a space to another address",
	RunE:  moveFunc,
}

// TODO: move all this to a separate client code
func moveFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	to, space := getMoveOp(args)
	cli := client.New(uri, requestTimeout)

	utx := &chain.MoveTx{
		BaseTx: &chain.BaseTx{},
		To:     to,
		Space:  space,
	}

	opts := []client.OpOption{client.WithPollTx()}
	_, cost, err := client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...)
	if err != nil {
		return err
	}

	addr := crypto.PubkeyToAddress(priv.PublicKey)
	b, err := cli.Balance(addr)
	if err != nil {
		return err
	}
	color.Cyan("Address=%s Balance=%d Cost=%d", addr, b, cost)
	return nil
}

func getMoveOp(args []string) (to common.Address, space string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "expected exactly 2 arguments, got %d", len(args))
		os.Exit(128)
	}

	addr := common.HexToAddress(args[0])
	if err := parser.CheckContents(args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse space %v", err)
		os.Exit(128)
	}
	return addr, args[1]
}
