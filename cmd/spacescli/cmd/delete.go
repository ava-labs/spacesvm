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
	Use:   "delete [options] <prefix/key>",
	Short: "Deletes a key-value pair for the given prefix",
	RunE:  deleteFunc,
}

// TODO: move all this to a separate client code
func deleteFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	space, key := getPathOp(args)
	cli := client.New(uri, requestTimeout)

	utx := &chain.DeleteTx{
		BaseTx: &chain.BaseTx{},
		Space:  space,
		Key:    key,
	}

	opts := []client.OpOption{client.WithPollTx(), client.WithInfo(space)}
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
