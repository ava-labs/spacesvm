// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"fmt"
	"os"

	"github.com/ava-labs/avalanchego/vms/rpcchainvm"
	"github.com/ava-labs/spacesvm/cmd/spacesvm/version"
	"github.com/ava-labs/spacesvm/vm"
	"github.com/hashicorp/go-plugin"
	log "github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
)

func init() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StreamHandler(os.Stderr, log.LogfmtFormat())))
}

var rootCmd = &cobra.Command{
	Use:        "spacesvm",
	Short:      "SpacesVM agent",
	SuggestFor: []string{"spacesvm"},
	RunE:       runFunc,
}

func init() {
	cobra.EnablePrefixMatching = true
}

func init() {
	rootCmd.AddCommand(
		version.NewCommand(),
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "spacesvm failed %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// TODO: serve separate endpoint for range query
// e.g., GET http://localhost/vm/foo returns "bar"
func runFunc(cmd *cobra.Command, args []string) error {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: rpcchainvm.Handshake,
		Plugins: map[string]plugin.Plugin{
			"vm": rpcchainvm.New(&vm.VM{}),
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
	return nil
}
