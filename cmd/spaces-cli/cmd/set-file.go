// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/parser"
	"github.com/ava-labs/spacesvm/tree"
)

var setFileCmd = &cobra.Command{
	Use:   "set-file [options] <space/key> <file path>",
	Short: "Writes a file to the given space",
	RunE:  setFileFunc,
}

func setFileFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	space, f, err := getSetFileOp(args)
	if err != nil {
		return err
	}
	defer f.Close()

	cli := client.New(uri, requestTimeout)
	g, err := cli.Genesis(context.Background())
	if err != nil {
		return err
	}

	// TODO: protect against overflow
	path, err := tree.Upload(context.Background(), cli, priv, space, f, int(g.MaxValueSize))
	if err != nil {
		return err
	}

	color.Green("uploaded file %s from %s", path, f.Name())
	return nil
}

func getSetFileOp(args []string) (space string, f *os.File, err error) {
	if len(args) != 2 {
		return "", nil, fmt.Errorf("expected exactly 2 arguments, got %d", len(args))
	}

	spaceKey := args[0]
	if err := parser.CheckContents(spaceKey); err != nil {
		return "", nil, fmt.Errorf("%w: failed to parse space", err)
	}

	filePath := args[1]
	if _, err := os.Stat(filePath); err != nil {
		return "", nil, fmt.Errorf("%w: file is not accessible", err)
	}

	f, err = os.Open(filePath)
	if err != nil {
		return "", nil, fmt.Errorf("%w: failed to open file", err)
	}

	return spaceKey, f, nil
}
