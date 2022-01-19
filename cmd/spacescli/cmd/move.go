// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"

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

func moveFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	to, space, err := getMoveOp(args)
	if err != nil {
		return err
	}

	utx := &chain.MoveTx{
		BaseTx: &chain.BaseTx{},
		To:     to,
		Space:  space,
	}

	cli := client.New(uri, requestTimeout)
	opts := []client.OpOption{client.WithPollTx()}
	if verbose {
		opts = append(opts, client.WithInfo(space))
		opts = append(opts, client.WithBalance())
	}
	if _, _, err := client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...); err != nil {
		return err
	}

	color.Green("moved %s to %s", space, to.Hex())
	return nil
}

func getMoveOp(args []string) (to common.Address, space string, err error) {
	if len(args) != 2 {
		return common.Address{}, "", fmt.Errorf("expected exactly 2 arguments, got %d", len(args))
	}

	addr := common.HexToAddress(args[0])
	if err := parser.CheckContents(args[1]); err != nil {
		return common.Address{}, "", fmt.Errorf("%w: failed to parse space", err)
	}
	return addr, args[1], nil
}
