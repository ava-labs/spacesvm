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
	workDir     string
	genesisFile string
)

// NewCommand implements "quark-cli" command.
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
	return cmd
}

func genesisFunc(cmd *cobra.Command, args []string) error {
	// Note: genesis block must have the min difficulty and block cost or else
	// the execution context logic may over/underflow
	blk := &chain.Block{
		Tmstmp:     time.Now().Unix(),
		Difficulty: chain.MinDifficulty,
		Cost:       chain.MinBlockCost,
	}
	b, err := chain.Marshal(blk)
	if err != nil {
		return err
	}
	if err := os.WriteFile(genesisFile, b, 0644); err != nil {
		return err
	}
	color.Green("created genesis and saved to %s", genesisFile)
	return nil
}
