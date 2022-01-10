// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// "quark-cli" implements quarkvm client operation interface.
package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/ava-labs/avalanchego/utils/crypto"
)

const (
	requestTimeout = 30 * time.Second
	fsModeWrite    = 0o600
)

var (
	privateKeyFile string
	uri            string
	workDir        string
	f              *crypto.FactorySECP256K1R

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
	f = &crypto.FactorySECP256K1R{}

	cobra.EnablePrefixMatching = true
	rootCmd.AddCommand(
		claimCmd,
		createCmd,
		deleteCmd,
		genesisCmd,
		getCmd,
		createCmd,
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
