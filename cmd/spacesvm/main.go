// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/ava-labs/avalanchego/vms/rpcchainvm"
	"github.com/ava-labs/spacesvm/cmd/spacesvm/version"
	"github.com/ava-labs/spacesvm/vm"
	log "github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
)

// All addresses on the C-Chain with > 2 transactions as of 1/15/22
// Hash: 0xccbf8e430b30d08b5b3342208781c40b373d1b5885c1903828f367230a2568da

//go:embed airdrops/011522.json
var AirdropData []byte

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
	rpcchainvm.Serve(&vm.VM{AirdropData: AirdropData})

	// Remove airdrop reference so VM can free memory
	AirdropData = nil

	return nil
}
