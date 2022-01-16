// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/tree"
)

var resolveFileCmd = &cobra.Command{
	Use:   "resolve-file [options] <space/key> <output path>",
	Short: "Reads a file at space/key and saves it to disk",
	RunE:  resolveFileFunc,
}

// TODO: move all this to a separate client code
func resolveFileFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "expected exactly 2 argument, got %d", len(args))
		os.Exit(128)
	}

	filePath := args[1]
	if _, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "file already exists %v", err)
		os.Exit(128)
	}

	f, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open file %v", err)
		os.Exit(128)
	}

	cli := client.New(uri, requestTimeout)
	return tree.Download(cli, args[0], f)
}
