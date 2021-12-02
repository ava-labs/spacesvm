// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package get implements "get" commands.
package get

import (
	"time"

	// "github.com/ava-labs/avalanchego/utils/rpc"
	// "github.com/fatih/color"
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
	// priv, err := create.LoadPK(privateKeyFile)
	// if err != nil {
	// 	return err
	// }

	// if !strings.HasPrefix(endpoint, "/") {
	// 	endpoint = "/" + endpoint
	// }
	// color.Blue("creating requester with URL %s and endpoint %q", url, endpoint)
	// _ = rpc.NewEndpointRequester(
	// 	url,
	// 	endpoint,
	// 	"quarkvm",
	// 	requestTimeout,
	// )

	return nil
}
