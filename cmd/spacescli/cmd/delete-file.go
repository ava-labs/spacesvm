// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/tree"
)

var deleteFileCmd = &cobra.Command{
	Use:   "delete-file [options] <space/key>",
	Short: "Deletes all hashes reachable from root file identifier",
	RunE:  deleteFileFunc,
}

// TODO: move all this to a separate client code
func deleteFileFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "expected exactly 2 argument, got %d", len(args))
		os.Exit(128)
	}

	cli := client.New(uri, requestTimeout)
	return tree.Delete(context.Background(), cli, args[0], priv)
}
