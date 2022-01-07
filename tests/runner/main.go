// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// runner uses "avalanche-network-runner" to set up a local network.
package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ava-labs/quarkvm/chain"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:        "runner",
	Short:      "avalanche-network-runner wrapper",
	SuggestFor: []string{"network-runner"},
	RunE:       runFunc,
}

func init() {
	cobra.EnablePrefixMatching = true
}

var (
	avalancheGoBinPath string
	vmID               string
	vmGenesisPath      string
	minDifficulty      uint64
	minBlockCost       uint64
	minExpiry          uint64
	pruneInterval      uint64
	outputPath         string
)

const defaultVMID = "tGas3T58KzdjLHhBDMnH2TvrddhqTji5iZAMZ3RXs2NLpSnhH"

func init() {
	f, err := ioutil.TempFile(os.TempDir(), "testrunnergenesis")
	if err != nil {
		fmt.Fprintf(os.Stderr, "runner failed create temp file %v\n", err)
		os.Exit(1)
	}
	genesisPath := f.Name()
	f.Close()

	rootCmd.PersistentFlags().StringVar(
		&avalancheGoBinPath,
		"avalanchego-path",
		"",
		"avalanchego binary path",
	)
	rootCmd.PersistentFlags().StringVar(
		&vmID,
		"vm-id",
		defaultVMID,
		"VM ID (must be formatted ids.ID)",
	)
	rootCmd.PersistentFlags().StringVar(
		&vmGenesisPath,
		"vm-genesis-path",
		genesisPath,
		"VM genesis file path",
	)
	rootCmd.PersistentFlags().Uint64Var(
		&minDifficulty,
		"min-difficulty",
		chain.DefaultMinDifficulty,
		"minimum difficulty for mining",
	)
	rootCmd.PersistentFlags().Uint64Var(
		&minBlockCost,
		"min-block-cost",
		chain.DefaultMinBlockCost,
		"minimum block cost",
	)
	rootCmd.PersistentFlags().Uint64Var(
		&minExpiry,
		"min-expiry",
		chain.DefaultMinExpiryTime,
		"minimum number of seconds to expire prefix since its block time",
	)
	rootCmd.PersistentFlags().Uint64Var(
		&pruneInterval,
		"prune-interval",
		chain.DefaultPruneInterval,
		"prune interval in seconds",
	)
	rootCmd.PersistentFlags().StringVar(
		&outputPath,
		"output-path",
		"",
		"output YAML path to write local cluster information",
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "runner failed %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
