// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// "spaces-cli" implements spacesvm client operation interface.
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
	verbose        bool
	workDir        string

	rootCmd = &cobra.Command{
		Use:        "spaces-cli",
		Short:      "SpacesVM CLI",
		SuggestFor: []string{"spaces-cli", "spacescli", "spacesctl"},
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
		resolveCmd,
		infoCmd,
		activityCmd,
		transferCmd,
		moveCmd,
		setFileCmd,
		resolveFileCmd,
		deleteFileCmd,
		networkCmd,
		ownedCmd,
	)

	rootCmd.PersistentFlags().StringVar(
		&privateKeyFile,
		"private-key-file",
		".spaces-cli-pk",
		"private key file path",
	)
	rootCmd.PersistentFlags().StringVar(
		&uri,
		"endpoint",
		"https://api.tryspaces.xyz",
		"RPC endpoint for VM",
	)
	rootCmd.PersistentFlags().BoolVar(
		&verbose,
		"verbose",
		false,
		"Print verbose information about operations",
	)
}

func Execute() error {
	return rootCmd.Execute()
}
