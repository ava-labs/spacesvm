// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
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
$ quark-cli claim hello.avax
<<COMMENT
success
COMMENT

# The existing prefix can be overwritten by a different owner.
# Once claimed, all existing key-value pairs are deleted.
$ quark-cli claim hello.avax --private-key-file=.different-key
<<COMMENT
success
COMMENT

# The prefix can be claimed if and only if
# the previous prefix (owner) info has not been expired.
# Even if the prefix is claimed by the same owner,
# all underlying key-values are deleted.
$ quark-cli claim hello.avax
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

	pfx := getClaimOp(args)
	cli := client.New(uri, requestTimeout)

	utx := &chain.ClaimTx{
		BaseTx: &chain.BaseTx{
			Pfx: pfx,
		},
	}

	opts := []client.OpOption{client.WithPollTx(), client.WithPrefixInfo(pfx)}
	_, err = client.SignIssueTx(context.Background(), cli, utx, priv, opts...)
	return err
}

func getClaimOp(args []string) (pfx []byte) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "expected exactly 1 argument, got %d", len(args))
		os.Exit(128)
	}

	pfx = []byte(args[0])
	if bytes.HasSuffix(pfx, []byte{'/'}) {
		pfx = pfx[:len(pfx)-1]
	}

	// check here first before parsing in case "pfx" is empty
	if err := parser.CheckPrefix(pfx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to verify prefix %v", err)
		os.Exit(128)
	}
	if _, _, _, err := parser.ParsePrefixKey(pfx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse prefix %v", err)
		os.Exit(128)
	}

	return pfx
}
