// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package claim

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/client"
	"github.com/ava-labs/quarkvm/cmd/quarkcli/create"
	"github.com/ava-labs/quarkvm/parser"
)

func init() {
	cobra.EnablePrefixMatching = true
}

var (
	privateKeyFile string
	uri            string
	requestTimeout time.Duration
	prefixInfo     bool
)

// NewCommand implements "quark-cli claim" command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
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
	cmd.PersistentFlags().StringVar(
		&privateKeyFile,
		"private-key-file",
		".quark-cli-pk",
		"private key file path",
	)
	cmd.PersistentFlags().StringVar(
		&uri,
		"endpoint",
		"http://127.0.0.1:9650",
		"RPC Endpoint for VM",
	)
	cmd.PersistentFlags().DurationVar(
		&requestTimeout,
		"request-timeout",
		30*time.Second,
		"timeout for transaction issuance and confirmation",
	)
	cmd.PersistentFlags().BoolVar(
		&prefixInfo,
		"prefix-info",
		true,
		"'true' to print out the prefix owner information",
	)
	return cmd
}

// TODO: move all this to a separate client code
func claimFunc(cmd *cobra.Command, args []string) error {
	priv, err := create.LoadPK(privateKeyFile)
	if err != nil {
		return err
	}
	pk, err := chain.FormatPK(priv.PublicKey())
	if err != nil {
		return err
	}

	pfx := getClaimOp(args)

	color.Blue("creating requester with URI %s for prefix %q", uri, pfx)
	cli := client.New(uri, requestTimeout)

	utx := &chain.ClaimTx{
		BaseTx: &chain.BaseTx{
			Sender: pk,
			Prefix: pfx,
		},
	}

	opts := []client.OpOption{client.WithPollTx()}
	if prefixInfo {
		opts = append(opts, client.WithPrefixInfo(pfx))
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	_, err = client.MineSignIssueTx(ctx, cli, utx, priv, opts...)
	cancel()
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
