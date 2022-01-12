// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/quarkvm/chain"
)

var (
	genesisFile string

	minDifficulty      int64
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
		&minDifficulty,
		"min-difficulty",
		-1,
		"minimum difficulty for mining",
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
	RunE:  genesisFunc,
}

func genesisFunc(cmd *cobra.Command, args []string) error {
	genesis := chain.DefaultGenesis()
	if minDifficulty >= 0 {
		genesis.MinDifficulty = uint64(minDifficulty)
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
