// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"fmt"
	"os"

	"github.com/ava-labs/quarkvm/cmd/quarkcli/claim"
	"github.com/ava-labs/quarkvm/cmd/quarkcli/create"
	"github.com/ava-labs/quarkvm/cmd/quarkcli/put"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:        "quark-cli",
	Short:      "QuarkVM client CLI",
	SuggestFor: []string{"quark-cli"},
}

func init() {
	cobra.EnablePrefixMatching = true
}

func init() {
	rootCmd.AddCommand(
		claim.NewCommand(),
		create.NewCommand(),
		put.NewCommand(),
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "quark-cli failed %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
