// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package get

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/quarkvm/client"
	"github.com/ava-labs/quarkvm/parser"
)

func init() {
	cobra.EnablePrefixMatching = true
}

var (
	privateKeyFile string
	url            string
	endpoint       string
	requestTimeout time.Duration
	limit          uint32
	withPrefix     bool
	prefixInfo     bool
)

// NewCommand implements "quark-cli get" command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [options] <prefix/key> <rangeEnd>",
		Short: "Reads the keys with the given prefix",
		Long: `
If no range end is given, it only reads the value for the
specified key if it exists. If a range end is given, it reads
all key-values in [start,end) at most "limit" entries.
If non-empty value is given, claim and write the given key to the store.

The prefix is automatically parsed with the delimiter "/".
When given a key "foo/hello", the "claim" creates the transaction
with "foo" as prefix and "hello" as key. The prefix/key cannot
have more than one delimiter (e.g., "foo/hello/world" is invalid)
in order to maintain the flat key space.

# If key and value are empty,
# then only issue "ClaimTx" for its ownership.
#
# "hello.avax" is the prefix (or namespace)
$ quark-cli claim hello.avax
<<COMMENT
success
COMMENT

# If the value is non-empty,
# then issue "SetTx" to update prefix info and write key-value pair.
#
# "hello.avax" is the prefix (or namespace)
# "foo" is the key
# "hello world" is the value
$ quark-cli claim hello.avax/foo1 "hello world 1"
$ quark-cli claim hello.avax/foo2 "hello world 2"
$ quark-cli claim hello.avax/foo3 "hello world 3"
<<COMMENT
success
COMMENT

# To read the existing key-value pair.
$ quark-cli get hello.avax/foo1
<<COMMENT
"hello.avax/foo1" "hello world 1"
COMMENT

# To read key-values with the prefix.
$ quark-cli get hello.avax/foo --with-prefix
<<COMMENT
"hello.avax/foo1" "hello world 1"
"hello.avax/foo2" "hello world 2"
"hello.avax/foo3" "hello world 3"
COMMENT

# To read key-values with the range end [start,end).
$ quark-cli get hello.avax/foo1 hello.avax/foo3
<<COMMENT
"hello.avax/foo1" "hello world 1"
"hello.avax/foo2" "hello world 2"
COMMENT

`,
		RunE: getFunc,
	}
	cmd.PersistentFlags().StringVar(
		&privateKeyFile,
		"private-key-file",
		".quark-cli-pk",
		"private key file path",
	)
	cmd.PersistentFlags().StringVar(
		&url,
		"url",
		"http://127.0.0.1:9650",
		"RPC URL for VM",
	)
	cmd.PersistentFlags().StringVar(
		&endpoint,
		"endpoint",
		"",
		"RPC endpoint for VM",
	)
	cmd.PersistentFlags().DurationVar(
		&requestTimeout,
		"request-timeout",
		30*time.Second,
		"timeout for transaction issuance and confirmation",
	)
	cmd.PersistentFlags().Uint32Var(
		&limit,
		"limit",
		0,
		"non-zero to limit the number of key-values in the response",
	)
	cmd.PersistentFlags().BoolVar(
		&withPrefix,
		"with-prefix",
		false,
		"'true' for prefix query",
	)
	cmd.PersistentFlags().BoolVar(
		&prefixInfo,
		"prefix-info",
		true,
		"'true' to print out the prefix owner information",
	)
	return cmd
}

// TODO: move all this to a separate client code
func getFunc(cmd *cobra.Command, args []string) error {
	pfx, key, rangeEnd := getGetOp(args, withPrefix)

	color.Blue("creating requester with URL %s and endpoint %q for prefix %q and key %q", url, endpoint, pfx, key)
	cli := client.New(url, endpoint, requestTimeout)

	opts := []client.OpOption{}
	if len(rangeEnd) > 0 {
		opts = append(opts, client.WithRangeEnd(rangeEnd))
	}
	if limit > 0 {
		opts = append(opts, client.WithRangeLimit(limit))
	}
	kvs, err := cli.Range(pfx, key, opts...)
	if err != nil {
		return err
	}

	// TODO: suppport custom output types (e.g., JSON)
	color.Green("range success %d key-values", len(kvs))
	for _, kv := range kvs {
		fmt.Printf("key: %q, value: %q\n", kv.Key, kv.Value)
	}

	if prefixInfo {
		info, err := cli.PrefixInfo(pfx)
		if err != nil {
			color.Red("cannot get prefix info %v", err)
		}
		color.Blue("prefix %q info %+v", pfx, info)
	}
	return nil
}

func getGetOp(args []string, withPrefix bool) (pfx []byte, key []byte, rangeEnd []byte) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "expected at least 1 arguments, got %d", len(args))
		os.Exit(128)
	}

	// [prefix/key] == "foo/bar"
	pfxKey := args[0]

	var err error
	pfx, key, rangeEnd, err = parser.ParsePrefixKey(
		[]byte(pfxKey),
		parser.WithCheckPrefix(),
		parser.WithCheckKey(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse prefix %v", err)
		os.Exit(128)
	}

	if !withPrefix {
		rangeEnd = nil
	}
	if len(args) > 1 {
		if withPrefix {
			fmt.Fprintf(os.Stderr, "--with-prefix cannot be used with range end")
			os.Exit(128)
		}
		rangeEnd = []byte(args[1])
	}
	return pfx, key, rangeEnd
}
