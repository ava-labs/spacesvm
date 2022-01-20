// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"errors"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create [options]",
	Short: "Creates a new key in the default location",
	Long: `
Creates a new key in the default location.
It will error if the key file already exists.

$ spaces-cli create

`,
	RunE: createFunc,
}

func createFunc(cmd *cobra.Command, args []string) error {
	if _, err := os.Stat(privateKeyFile); err == nil {
		// Already found, remind the user they have it
		priv, err := crypto.LoadECDSA(privateKeyFile)
		if err != nil {
			return err
		}
		color.Green("ABORTING!!! key for %s already exists at %s", crypto.PubkeyToAddress(priv.PublicKey), privateKeyFile)
		return os.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// Generate new key and save to disk
	// TODO: encrypt key
	priv, err := crypto.GenerateKey()
	if err != nil {
		return err
	}
	if err := crypto.SaveECDSA(privateKeyFile, priv); err != nil {
		return err
	}
	color.Green("created address %s and saved to %s", crypto.PubkeyToAddress(priv.PublicKey), privateKeyFile)
	return nil
}
