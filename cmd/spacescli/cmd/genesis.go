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

	minPrice           int64
	minBlockCost       int64
	claimReward        int64
	lifelineUnitReward int64

	magic uint64
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
		&minBlockCost,
		"min-block-cost",
		-1,
		"minimum block cost",
	)
	genesisCmd.PersistentFlags().Int64Var(
		&claimReward,
		"claim-reward",
		-1,
		"seconds until a prefix will expire after being claimed",
	)
	genesisCmd.PersistentFlags().Int64Var(
		&lifelineUnitReward,
		"lifeline-unit-reward",
		-1,
		"seconds per unit of fee that will be rewarded in a lifeline transaction",
	)
}

var genesisCmd = &cobra.Command{
	Use:   "genesis [magic] [allocations file] [options]",
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
	if minBlockCost >= 0 {
		genesis.MinBlockCost = uint64(minBlockCost)
	}
	if claimReward >= 0 {
		genesis.ClaimReward = uint64(claimReward)
	}
	if lifelineUnitReward >= 0 {
		genesis.LifelineUnitReward = uint64(lifelineUnitReward)
	}

	a, err := os.ReadFile(args[1])
	if err != nil {
		return err
	}
	allocs := []*chain.Allocation{}
	if err := json.Unmarshal(a, &allocs); err != nil {
		return err
	}
	// Store hash instead
	genesis.Allocations = allocs

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
