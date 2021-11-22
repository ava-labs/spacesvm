package claim

import (
	"fmt"
	"os"

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
		Run: claimFunc,
	}
	return cmd
}

func claimFunc(cmd *cobra.Command, args []string) {
	p := getClaimOp(args)
	fmt.Println("NOT IMPLEMENTED YET:", p)
}

func getClaimOp(args []string) (prefix string) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "expected one argument, got %d\n", len(args))
		os.Exit(128)
	}

	return args[0]
}
