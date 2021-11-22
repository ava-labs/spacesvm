package claim

import (
	"fmt"
	"os"

	"github.com/ava-labs/quarkvm/cmd/quarkcli/create"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	cobra.EnablePrefixMatching = true
}

// NewCommand implements "quark-cli" command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim [options] <prefix>",
		Short: "Claims the given prefix",
		Long: `
Claims the given prefix.

# "jim" is the prefix
$ quark-cli claim jim

`,
		RunE: claimFunc,
	}
	return cmd
}

func claimFunc(cmd *cobra.Command, args []string) error {
	pk, err := create.LoadPK()
	if err != nil {
		return err
	}
	color.Green("loaded address %s", pk.PublicKey().Address())
	p := getClaimOp(args)
	fmt.Println("NOT IMPLEMENTED YET:", p)
	return nil
}

func getClaimOp(args []string) (prefix string) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "expected one argument, got %d\n", len(args))
		os.Exit(128)
	}

	return args[0]
}
