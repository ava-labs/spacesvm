// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
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

	space, units, err := getLifelineOp(args)
	if err != nil {
		return err
	}

	utx := &chain.LifelineTx{
		BaseTx: &chain.BaseTx{},
		Space:  space,
		Units:  units,
	}

	cli := client.New(uri, requestTimeout)
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

func getLifelineOp(args []string) (space string, units uint64, err error) {
	if len(args) != 2 {
		return "", 0, fmt.Errorf("expected exactly 1 argument, got %d", len(args))
	}

	space = args[0]
	splits := strings.Split(space, "/")
	space = splits[0]

	// check here first before parsing in case "space" is empty
	if err := parser.CheckContents(space); err != nil {
		return "", 0, fmt.Errorf("%w: failed to verify space", err)
	}

	units, err = strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("%w: failed to parse units", err)
	}

	return space, units, nil
}
