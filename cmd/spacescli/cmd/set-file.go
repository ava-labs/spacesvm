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

	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/parser"
	"github.com/ava-labs/spacesvm/tree"
)

var setFileCmd = &cobra.Command{
	Use:   "set-file [options] <space/key> <file path>",
	Short: "Writes a file to the given space",
	RunE:  setFileFunc,
}

// TODO: move all this to a separate client code
func setFileFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	space, f := getSetFileOp(args)
	defer f.Close()

	cli := client.New(uri, requestTimeout)
	g, err := cli.Genesis()
	if err != nil {
		return err
	}

	if _, err := tree.Upload(context.Background(), cli, priv, space, f, g.MaxValueSize); err != nil {
		return err
	}

	addr := crypto.PubkeyToAddress(priv.PublicKey)
	b, err := cli.Balance(addr)
	if err != nil {
		return err
	}
	color.Cyan("Address=%s Balance=%d", addr, b)
	return nil
}

func getSetFileOp(args []string) (space string, f *os.File) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "expected exactly 2 arguments, got %d", len(args))
		os.Exit(128)
	}

	spaceKey := args[0]
	if err := parser.CheckContents(spaceKey); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse space %v", err)
		os.Exit(128)
	}

	filePath := args[1]
	if _, err := os.Stat(filePath); err != nil {
		fmt.Fprintf(os.Stderr, "file is not accessible %v", err)
		os.Exit(128)
	}

	f, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open file %v", err)
		os.Exit(128)
	}

	return spaceKey, f
}
