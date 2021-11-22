// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package get implements "get" commands.
package get

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ava-labs/quarkvm/cmd/quarkcli/create"
	"github.com/ava-labs/quarkvm/transaction"
	"github.com/ava-labs/quarkvm/vm"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	cobra.EnablePrefixMatching = true
}

var (
	privateKeyFile string
	url            string
	endpoint       string
	requestTimeout time.Duration
)

// NewCommand implements "quark-cli" command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [options] <key> <rangeEnd>",
		Short: "Reads the given key or range from the store",
		Long: `
Reads the given key or range from the store.

$ quark-cli get jim/foo

`,
		RunE: getFunc,
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
	return cmd
}

// TODO: move all this to a separate client code
func getFunc(cmd *cobra.Command, args []string) error {
	priv, err := create.LoadPK(privateKeyFile)
	if err != nil {
		return err
	}

	k, rangeEnd := getGetOp(args)

	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	color.Blue("creating requester with URL %s and endpoint %q", url, endpoint)
	requester := rpc.NewEndpointRequester(
		url,
		endpoint,
		"quarkvm",
		requestTimeout,
	)

	// create unsigned transaction
	// don't string case pubkey
	// after grpc hop, 32 bytes becomes 64, causing
	// panic: ed25519: bad public key length: 64
	utx := transaction.Unsigned{
		PublicKey: priv.PublicKey().Bytes(),
		Op:        "Range",
		Key:       k,
		RangeEnd:  rangeEnd,
	}

	// sign the unsigned transaction
	sig, err := priv.Sign(utx.Bytes())
	if err != nil {
		return err
	}

	// create transaction
	tx := &transaction.Transaction{
		Unsigned:  utx,
		Signature: sig,
	}

	// issue the transaction over tx
	color.Yellow("sending range [%q, %q]", k, rangeEnd)
	resp := new(vm.IssueTxReply)
	if err := requester.SendRequest(
		"issueTx",
		&vm.IssueTxArgs{Transaction: tx},
		resp,
	); err != nil {
		color.Red("range failed %v", err)
		return err
	}
	if !resp.Success {
		return fmt.Errorf("range %q failed (%v)", k, resp.Error)
	}
	color.Green("range [%q, %q] success %v", k, rangeEnd, resp.Success)

	// TODO: configurable output format
	for _, kv := range resp.RangeResponse.KeyValues {
		fmt.Println(string(kv.Key), string(kv.Value))
	}
	return nil
}

func getGetOp(args []string) (key string, rangeEnd string) {
	if len(args) > 2 {
		fmt.Fprintf(os.Stderr, "expected at most 2 arguments, got %d\n", len(args))
		os.Exit(128)
	}
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "expected at least 2 arguments, got %d\n", len(args))
		os.Exit(128)
	}
	if len(args) == 1 {
		return args[0], ""
	}
	return args[0], args[1]
}
