// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [options] <space/key>",
	Short: "Deletes a key-value pair for the given space",
	RunE:  deleteFunc,
}

func deleteFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	space, key, err := getPathOp(args)
	if err != nil {
		return err
	}

	utx := &chain.DeleteTx{
		BaseTx: &chain.BaseTx{},
		Space:  space,
		Key:    key,
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

	color.Green("deleted %s from %s", key, space)
	return nil
}
