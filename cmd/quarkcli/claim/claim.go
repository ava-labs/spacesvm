// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package claim

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/quarkvm/chain"
	client "github.com/ava-labs/quarkvm/client/v0alpha"
	"github.com/ava-labs/quarkvm/cmd/quarkcli/create"
	"github.com/ava-labs/quarkvm/storage"
)

func init() {
	cobra.EnablePrefixMatching = true
}

var (
	privateKeyFile string
	url            string
	endpoint       string
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
		&url,
		"url",
		"http://127.0.0.1:9650",
		"RPC URL for VM",
	)
	cmd.PersistentFlags().StringVar(
		&endpoint,
		"endpoint",
		"",
		"RPC endpoint for VM",
	)
	cmd.PersistentFlags().DurationVar(
		&requestTimeout,
		"request-timeout",
		30*time.Second,
		"set it to 0 to not wait for transaction confirmation",
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

	pfx := getClaimOp(args)

	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	color.Blue("creating requester with URL %s and endpoint %q for prefix %q", url, endpoint, pfx)
	cli := client.New(url, endpoint, requestTimeout)

	utx := &chain.ClaimTx{
		BaseTx: &chain.BaseTx{
			Sender: priv.PublicKey().Bytes(),
			Prefix: pfx,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	mtx, err := cli.Mine(ctx, utx)
	cancel()
	if err != nil {
		return err
	}

	b, err := chain.UnsignedBytes(mtx)
	if err != nil {
		return err
	}
	sig, err := priv.Sign(b)
	if err != nil {
		return err
	}
	tx := chain.NewTx(mtx, sig)
	if err := tx.Init(); err != nil {
		return err
	}

	color.Yellow("issuing tx %s with block ID %s", tx.ID(), mtx.GetBlockID())
	txID, err := cli.IssueTx(tx.Bytes())
	if err != nil {
		return err
	}

	color.Green("issued transaction %s (now polling)", txID)
	ctx, cancel = context.WithTimeout(context.Background(), requestTimeout)
	confirmed, err := cli.PollTx(ctx, txID)
	cancel()
	if err != nil {
		return err
	}
	if confirmed {
		color.Green("transaction %s confirmed", txID)
	} else {
		color.Yellow("transaction %s not confirmed", txID)
	}

	if prefixInfo {
		info, err := cli.PrefixInfo(pfx)
		if err != nil {
			color.Red("cannot get prefix info %v", err)
		}
		color.Blue("prefix %q info %+v", pfx, info)
	}
	return nil
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

	if err := storage.CheckPrefix(pfx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to verify prefix %v", err)
		os.Exit(128)
	}
	if _, _, _, err := storage.ParsePrefixKey(pfx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse prefix %v", err)
		os.Exit(128)
	}

	return pfx
}
