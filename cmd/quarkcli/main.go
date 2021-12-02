// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// "quark-cli" implements quarkvm client operation interface.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ava-labs/quarkvm/cmd/quarkcli/create"
	"github.com/ava-labs/quarkvm/cmd/quarkcli/get"
	"github.com/ava-labs/quarkvm/cmd/quarkcli/put"
)

var rootCmd = &cobra.Command{
	Use:        "quark-cli",
	Short:      "QuarkVM client CLI",
	SuggestFor: []string{"quark-cli", "quarkcli", "quarkctl"},
}

func init() {
	cobra.EnablePrefixMatching = true
}

func init() {
	rootCmd.AddCommand(
		create.NewCommand(),
		put.NewCommand(),
		get.NewCommand(),
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "quark-cli failed %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
