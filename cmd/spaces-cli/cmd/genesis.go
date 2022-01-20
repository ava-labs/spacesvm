// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/chain"
)

var (
	genesisFile string
	magic       uint64

	minPrice    int64
	claimReward int64

	airdropHash  string
	airdropUnits uint64
)

func init() {
	genesisCmd.PersistentFlags().StringVar(
		&genesisFile,
		"genesis-file",
		filepath.Join(workDir, "genesis.json"),
		"genesis file path",
	)
	genesisCmd.PersistentFlags().Int64Var(
		&minPrice,
		"min-price",
		-1,
		"minimum price",
	)
	genesisCmd.PersistentFlags().Int64Var(
		&claimReward,
		"claim-reward",
		-1,
		"seconds until a spaces will expire after being claimed",
	)
	genesisCmd.PersistentFlags().StringVar(
		&airdropHash,
		"airdrop-hash",
		"",
		"hash of airdrop data",
	)
	genesisCmd.PersistentFlags().Uint64Var(
		&airdropUnits,
		"airdrop-units",
		0,
		"units to allocate to each airdrop address",
	)
}

var genesisCmd = &cobra.Command{
	Use:   "genesis [magic] [custom allocations file] [options]",
	Short: "Creates a new genesis in the default location",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("invalid args")
		}

		m, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return err
		}
		magic = m
		if magic == 0 {
			return chain.ErrInvalidMagic
		}

		return nil
	},
	RunE: genesisFunc,
}

func genesisFunc(cmd *cobra.Command, args []string) error {
	genesis := chain.DefaultGenesis()
	genesis.Magic = magic
	if minPrice >= 0 {
		genesis.MinPrice = uint64(minPrice)
	}
	if claimReward >= 0 {
		genesis.ClaimReward = uint64(claimReward)
	}
	if len(airdropHash) > 0 {
		genesis.AirdropHash = airdropHash
		if airdropUnits == 0 {
			return errors.New("non-zero airdrop units required")
		}
		genesis.AirdropUnits = airdropUnits
	}

	a, err := os.ReadFile(args[1])
	if err != nil {
		return err
	}
	allocs := []*chain.CustomAllocation{}
	if err := json.Unmarshal(a, &allocs); err != nil {
		return err
	}
	genesis.CustomAllocation = allocs

	b, err := json.Marshal(genesis)
	if err != nil {
		return err
	}
	if err := os.WriteFile(genesisFile, b, fsModeWrite); err != nil {
		return err
	}
	color.Green("created genesis and saved to %s", genesisFile)
	return nil
}
