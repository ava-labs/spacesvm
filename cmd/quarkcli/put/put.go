package put

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
		Use:   "put [options] <prefix> <key> <value>",
		Short: "Puts the given key-value pair into the store",
		Long: `
Puts the given key into the store.

# "jim" is the prefix
# "foo" is the key
# "hello world" is the value
$ quark-cli put jim foo "hello world"

`,
		Run: putFunc,
	}
	return cmd
}

func putFunc(cmd *cobra.Command, args []string) {
	p, k, v := getPutOp(args)
	fmt.Println("NOT IMPLEMENTED YET:", p, k, v)
}

func getPutOp(args []string) (prefix string, key string, value string) {
	if len(args) != 3 {
		fmt.Fprintf(os.Stderr, "expected three arguments, got %d\n", len(args))
		os.Exit(128)
	}

	return args[0], args[1], args[2]
}
