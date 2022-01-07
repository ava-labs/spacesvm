// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package genesis

import (
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/quarkvm/chain"
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
	workDir       string
	genesisFile   string
	minDifficulty uint64
	minBlockCost  uint64
	minExpiry     uint64
	pruneInterval uint64
)

// NewCommand implements "quark-cli genesis" command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "genesis [options]",
		Short: "Creates a new genesis in the default location",
		RunE:  genesisFunc,
	}
	cmd.PersistentFlags().StringVar(
		&genesisFile,
		"genesis-file",
		filepath.Join(workDir, "genesis.json"),
		"genesis file path",
	)
	cmd.PersistentFlags().Uint64Var(
		&minDifficulty,
		"min-difficulty",
		chain.DefaultMinDifficulty,
		"minimum difficulty for mining",
	)
	cmd.PersistentFlags().Uint64Var(
		&minBlockCost,
		"min-block-cost",
		chain.DefaultMinBlockCost,
		"minimum block cost",
	)
	cmd.PersistentFlags().Uint64Var(
		&minExpiry,
		"min-expiry",
		chain.DefaultMinExpiryTime,
		"minimum number of seconds to expire prefix since its block time",
	)
	cmd.PersistentFlags().Uint64Var(
		&pruneInterval,
		"prune-interval",
		chain.DefaultPruneInterval,
		"prune interval in seconds",
	)
	return cmd
}

const fsModeWrite = 0o600

func genesisFunc(cmd *cobra.Command, args []string) error {
	g := chain.Genesis{
		MinDifficulty: minDifficulty,
		MinBlockCost:  minBlockCost,
		MinExpiry:     minExpiry,
		PruneInterval: pruneInterval,
	}
	extraData, err := chain.Marshal(g)
	if err != nil {
		return err
	}
	// Note: genesis block must have the min difficulty and block cost or else
	// the execution context logic may over/underflow
	blk := &chain.StatefulBlock{
		Tmstmp:     time.Now().Unix(),
		Difficulty: minDifficulty,
		Cost:       minBlockCost,
		ExtraData:  extraData,
	}
	b, err := chain.Marshal(blk)
	if err != nil {
		return err
	}
	if err := os.WriteFile(genesisFile, b, fsModeWrite); err != nil {
		return err
	}
	color.Green("created genesis and saved to %s", genesisFile)
	return nil
}
