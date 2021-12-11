// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package put implements "put" commands.
package put

import (
	// "context"

	// "strings"
	"time"

	// "github.com/ava-labs/avalanchego/utils/rpc"
	// "github.com/fatih/color"
	"github.com/spf13/cobra"
	// "github.com/ava-labs/quarkvm/chain"
	// "github.com/ava-labs/quarkvm/cmd/quarkcli/create"
	// "github.com/ava-labs/quarkvm/vm"
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
		Use:   "put [options] <key> <value>",
		Short: "Puts the given key-value pair into the store",
		Long: `
Puts the given key into the store.

# prefix will be automatically parsed with delimiter "/"
# "jim" is the prefix (namespace)
# "foo" is the key
# "hello world" is the value
$ quark-cli put jim/foo "hello world"

`,
		RunE: putFunc,
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
func putFunc(cmd *cobra.Command, args []string) error {
	// TODO
	return nil
}
