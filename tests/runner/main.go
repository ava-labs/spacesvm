// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// runner uses "avalanche-network-runner" to set up a local network.
package main

import (
	"fmt"
	"os"

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
	outputPath         string
)

const defaultVMID = "tGas3T58KzdjLHhBDMnH2TvrddhqTji5iZAMZ3RXs2NLpSnhH"

func init() {
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
		"",
		"VM genesis file path",
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
