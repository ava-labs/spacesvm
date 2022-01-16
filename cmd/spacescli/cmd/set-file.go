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

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/parser"
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

	space, key, val := getSetFileOp(args)
	cli := client.New(uri, requestTimeout)

	utx := &chain.SetTx{
		BaseTx: &chain.BaseTx{},
		Space:  space,
		Key:    key,
		Value:  val,
	}

	opts := []client.OpOption{client.WithPollTx(), client.WithInfo(space)}
	_, err = client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...)
	if err != nil {
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

func getSetFileOp(args []string) (space string, key string, val []byte) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "expected exactly 2 arguments, got %d", len(args))
		os.Exit(128)
	}

	// [space/key] == "foo/bar"
	spaceKey := args[0]

	var err error
	space, key, err = parser.ResolvePath(spaceKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse prefix %v", err)
		os.Exit(128)
	}

	val = []byte(args[1])

	return space, key, val
}
