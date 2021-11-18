// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// quarkvmctl is a set of quarkvm client commands
// to interact with KVVM servers.
package main

import (
	"fmt"
	"os"

	"github.com/ava-labs/quarkvm/cmd/quarkvmctl/put"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:        "quarkvmctl",
	Short:      "Quark KVVM client CLI",
	SuggestFor: []string{"quarkvm-ctl"},
}

func init() {
	cobra.EnablePrefixMatching = true
}

func init() {
	rootCmd.AddCommand(
		put.NewCommand(),
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "quarkvmctl failed %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
