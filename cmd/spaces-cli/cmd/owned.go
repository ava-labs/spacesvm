// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/client"
)

var ownedCmd = &cobra.Command{
	Use:   "owned [options]",
	Short: "Fetches all owned spaces for the address associated with the private key",
	RunE:  ownedFunc,
}

func ownedFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}
	sender := crypto.PubkeyToAddress(priv.PublicKey)

	cli := client.New(uri, requestTimeout)
	spaces, err := cli.Owned(context.Background(), sender)
	if err != nil {
		return err
	}

	color.Green("address %s owns %+v", sender.Hex(), spaces)
	return nil
}
