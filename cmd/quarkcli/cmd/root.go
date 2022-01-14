// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// "quark-cli" implements spacesvm client operation interface.
package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"
)

const (
	requestTimeout = 30 * time.Second
	fsModeWrite    = 0o600
)

var (
	privateKeyFile string
	uri            string
	workDir        string

	rootCmd = &cobra.Command{
		Use:        "quark-cli",
		Short:      "QuarkVM client CLI",
		SuggestFor: []string{"quark-cli", "quarkcli", "quarkctl"},
	}
)

func init() {
	p, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	workDir = p

	cobra.EnablePrefixMatching = true
	rootCmd.AddCommand(
		createCmd,
		genesisCmd,
		claimCmd,
		lifelineCmd,
		setCmd,
		deleteCmd,
		getCmd,
	)

	rootCmd.PersistentFlags().StringVar(
		&privateKeyFile,
		"private-key-file",
		".quark-cli-pk",
		"private key file path",
	)
	rootCmd.PersistentFlags().StringVar(
		&uri,
		"endpoint",
		"http://127.0.0.1:9650",
		"RPC Endpoint for VM",
	)
}

func Execute() error {
	return rootCmd.Execute()
}
