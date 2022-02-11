// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/tree"
)

var resolveFileCmd = &cobra.Command{
	Use:   "resolve-file [options] <space/key> <output path>",
	Short: "Reads a file at space/key and saves it to disk",
	RunE:  resolveFileFunc,
}

func resolveFileFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("expected exactly 2 argument, got %d", len(args))
	}

	filePath := args[1]
	if _, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("file %s already exists", filePath)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s", filePath)
	}
	defer f.Close()

	cli := client.New(uri, requestTimeout)
	if err := tree.Download(context.Background(), cli, args[0], f); err != nil {
		return err
	}

	color.Green("resolved file %s and stored at %s", args[0], filePath)
	return nil
}
