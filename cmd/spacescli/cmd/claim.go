// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/parser"
)

var claimCmd = &cobra.Command{
	Use:   "claim [options] <space>",
	Short: "Claims the given space",
	Long: `
Claims the given space by issuing claim transaction
with the space information.

# Issues "ClaimTx" for the ownership of "hello.avax".
# "hello.avax" is the space (or namespace)
$ spaces-cli claim hello.avax
<<COMMENT
success
COMMENT
`,
	RunE: claimFunc,
}

func claimFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	space, err := getClaimOp(args)
	if err != nil {
		return err
	}

	utx := &chain.ClaimTx{
		BaseTx: &chain.BaseTx{},
		Space:  space,
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

	color.Green("claimed %s", space)
	return nil
}

func getClaimOp(args []string) (space string, err error) {
	if len(args) != 1 {
		return "", fmt.Errorf("expected exactly 1 argument, got %d", len(args))
	}

	space = args[0]
	splits := strings.Split(space, "/")
	space = splits[0]

	// check here first before parsing in case "space" is empty
	if err := parser.CheckContents(space); err != nil {
		return "", fmt.Errorf("%w: failed to verify space", err)
	}

	return space, nil
}
