// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/quarkvm/chain"
)

var (
	genesisFile string

	minPrice           int64
	minBlockCost       int64
	claimReward        int64
	lifelineUnitReward int64
	beneficiaryReward  int64
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
	genesisCmd.PersistentFlags().Int64Var(
		&beneficiaryReward,
		"beneficiary-reward",
		-1,
		"seconds added to the lifetime of a beneficiary prefix when a block is produced",
	)
}

var genesisCmd = &cobra.Command{
	Use:   "genesis [options]",
	Short: "Creates a new genesis in the default location",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("missing allocations file")
		}
		return nil
	},
	RunE: genesisFunc,
}

func genesisFunc(cmd *cobra.Command, args []string) error {
	genesis := chain.DefaultGenesis()
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
	if beneficiaryReward >= 0 {
		genesis.BeneficiaryReward = uint64(beneficiaryReward)
	}

	a, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	allocs := []*chain.Allocation{}
	if err := json.Unmarshal(a, &allocs); err != nil {
		return err
	}
	// Store hash instead
	genesis.Allocations = allocs

	b, err := chain.Marshal(genesis)
	if err != nil {
		return err
	}
	if err := os.WriteFile(genesisFile, b, fsModeWrite); err != nil {
		return err
	}
	color.Green("created genesis and saved to %s", genesisFile)
	return nil
}
