// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package create implements "create" commands.
package create

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ava-labs/quarkvm/crypto"
	"github.com/ava-labs/quarkvm/crypto/ed25519"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	p, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	workDir = p

	cobra.EnablePrefixMatching = true
}

var (
	workDir        string
	privateKeyFile string
)

// NewCommand implements "quark-cli" command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [options]",
		Short: "Creates a new key in the default location",
		Long: `
Creates a new key in the default location.
It will error if the key file already exists.

$ quark-cli create

`,
		RunE: createFunc,
	}
	cmd.PersistentFlags().StringVar(
		&privateKeyFile,
		"private-key-file",
		filepath.Join(workDir, ".quark-cli-pk"),
		"private key file path",
	)
	return cmd
}

func createFunc(cmd *cobra.Command, args []string) error {
	if _, err := os.Stat(privateKeyFile); err == nil {
		return fmt.Errorf("file already exists at %s", privateKeyFile)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// Generate new key and save to disk
	// TODO: encrypt key
	pk, err := ed25519.NewPrivateKey()
	if err != nil {
		return err
	}
	if err := os.WriteFile(privateKeyFile, pk.Bytes(), 0644); err != nil {
		return err
	}
	color.Green("created address %s and saved to %s", pk.PublicKey().Address(), privateKeyFile)
	return nil
}

// TODO: run before all functions (erroring if can't load)
func LoadPK(privPath string) (crypto.PrivateKey, error) {
	pk, err := os.ReadFile(privPath)
	if err != nil {
		return nil, err
	}
	return ed25519.LoadPrivateKey(pk)
}
