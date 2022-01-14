// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/client"
)

var getCmd = &cobra.Command{
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
$ spaces-cli claim hello.avax
<<COMMENT
success
COMMENT

# If the value is non-empty,
# then issue "SetTx" to update prefix info and write key-value pair.
#
# "hello.avax" is the prefix (or namespace)
# "foo" is the key
# "hello world" is the value
$ spaces-cli claim hello.avax/foo1 "hello world 1"
$ spaces-cli claim hello.avax/foo2 "hello world 2"
$ spaces-cli claim hello.avax/foo3 "hello world 3"
<<COMMENT
success
COMMENT

# To read the existing key-value pair.
$ spaces-cli get hello.avax/foo1
<<COMMENT
"hello.avax/foo1" "hello world 1"
COMMENT

# To read key-values with the prefix.
$ spaces-cli get hello.avax/foo --with-prefix
<<COMMENT
"hello.avax/foo1" "hello world 1"
"hello.avax/foo2" "hello world 2"
"hello.avax/foo3" "hello world 3"
COMMENT

# To read key-values with the range end [start,end).
$ spaces-cli get hello.avax/foo1 hello.avax/foo3
<<COMMENT
"hello.avax/foo1" "hello world 1"
"hello.avax/foo2" "hello world 2"
COMMENT

`,
	RunE: getFunc,
}

// TODO: move all this to a separate client code
func getFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "expected exactly 1 argument, got %d", len(args))
		os.Exit(128)
	}
	cli := client.New(uri, requestTimeout)
	_, v, err := cli.Resolve(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("%s=>%q\n", args[0], v)
	return nil
}
