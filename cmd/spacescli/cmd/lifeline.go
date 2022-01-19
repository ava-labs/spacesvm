// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/parser"
)

var lifelineCmd = &cobra.Command{
	Use:   "lifeline [options] <space> <units>",
	Short: "Extends the life of a given space",
	RunE:  lifelineFunc,
}

func lifelineFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	space, units := getLifelineOp(args)
	cli := client.New(uri, requestTimeout)

	utx := &chain.LifelineTx{
		BaseTx: &chain.BaseTx{},
		Space:  space,
		Units:  units,
	}

	opts := []client.OpOption{client.WithPollTx()}
	if verbose {
		opts = append(opts, client.WithInfo(space))
		opts = append(opts, client.WithBalance())
	}
	if _, _, err := client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...); err != nil {
		return err
	}

	color.Green("extended life of %s by %d units", space, units)
	return nil
}

func getLifelineOp(args []string) (space string, units uint64) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "expected exactly 1 argument, got %d", len(args))
		os.Exit(128)
	}

	space = args[0]
	splits := strings.Split(space, "/")
	space = splits[0]

	// check here first before parsing in case "space" is empty
	if err := parser.CheckContents(space); err != nil {
		fmt.Fprintf(os.Stderr, "failed to verify space %v", err)
		os.Exit(128)
	}

	units, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse units %v", err)
		os.Exit(128)
	}

	return space, units
}
