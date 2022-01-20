// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/tree"
)

var deleteFileCmd = &cobra.Command{
	Use:   "delete-file [options] <space/key>",
	Short: "Deletes all hashes reachable from root file identifier",
	RunE:  deleteFileFunc,
}

func deleteFileFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	if len(args) != 1 {
		return fmt.Errorf("expected exactly 1 argument, got %d", len(args))
	}

	cli := client.New(uri, requestTimeout)
	if err := tree.Delete(context.Background(), cli, args[0], priv); err != nil {
		return err
	}

	color.Green("deleted file %s", args[0])
	return nil
}
