// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/parser"
)

var claimCmd = &cobra.Command{
	Use:   "claim [options] <prefix>",
	Short: "Claims the given prefix",
	Long: `
Claims the given prefix by issuing claim transaction
with the prefix information.

# Issues "ClaimTx" for the ownership of "hello.avax".
# "hello.avax" is the prefix (or namespace)
$ spaces-cli claim hello.avax
<<COMMENT
success
COMMENT

# The existing prefix can be overwritten by a different owner.
# Once claimed, all existing key-value pairs are deleted.
$ spaces-cli claim hello.avax --private-key-file=.different-key
<<COMMENT
success
COMMENT

# The prefix can be claimed if and only if
# the previous prefix (owner) info has not been expired.
# Even if the prefix is claimed by the same owner,
# all underlying key-values are deleted.
$ spaces-cli claim hello.avax
<<COMMENT
success
COMMENT

`,
	RunE: claimFunc,
}

// TODO: move all this to a separate client code
func claimFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	space := getClaimOp(args)
	cli := client.New(uri, requestTimeout)

	utx := &chain.ClaimTx{
		BaseTx: &chain.BaseTx{},
		Space:  space,
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

func getClaimOp(args []string) (space string) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "expected exactly 1 argument, got %d", len(args))
		os.Exit(128)
	}

	space = args[0]
	splits := strings.Split(space, "/")
	space = splits[0]

	// check here first before parsing in case "pfx" is empty
	if err := parser.CheckContents(space); err != nil {
		fmt.Fprintf(os.Stderr, "failed to verify prefix %v", err)
		os.Exit(128)
	}

	return space
}
