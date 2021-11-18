package put

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	cobra.EnablePrefixMatching = true
}

// NewCommand implements "aws-k8s-tester eks" command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		// TODO: support "value" from the stdin
		Use:   "put [options] <key> <value>",
		Short: "Puts the given key-value pair into the store",
		Long: `
Puts the given key into the store.

# "foo" is the key
# "hello world" is the value
$ quarkvmctl put foo "hello world"

`,
		Run: putFunc,
	}
	return cmd
}

func putFunc(cmd *cobra.Command, args []string) {
	k, v := getPutOp(args)
	fmt.Println("NOT IMPLEMENTED YET:", k, v)
}

func getPutOp(args []string) (key string, value string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "expected two arguments, got %d\n", len(args))
		os.Exit(128)
	}

	key, value = args[0], args[1]
	return key, value
}
