// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"

	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/client"
	"github.com/ava-labs/quarkvm/parser"
)

var setCmd = &cobra.Command{
	Use:   "set [options] <prefix/key> <value>",
	Short: "Writes a key-value pair for the given prefix",
	Long: `
Issues "SetTx" to write a key-value pair.

The prefix is automatically parsed with the delimiter "/".
When given a key "foo/hello", the "set" creates the transaction
with "foo" as prefix and "hello" as key. The prefix/key cannot
have more than one delimiter (e.g., "foo/hello/world" is invalid)
in order to maintain the flat key space.

It assumes the prefix is already claimed via "quark-cli claim".
Otherwise, the set transaction will fail.

# claims the prefix "hello.avax"
# "hello.avax" is the prefix (or namespace)
$ quark-cli claim hello.avax
<<COMMENT
success
COMMENT

# writes a key-value pair for the given namespace (prefix)
# by issuing "SetTx" preceded by "IssueTx" on the prefix:
# "hello.avax" is the prefix (or namespace)
# "foo" is the key
# "hello world" is the value
$ quark-cli set hello.avax/foo "hello world"
<<COMMENT
success
COMMENT

# The existing key-value cannot be overwritten by a different owner.
# The prefix must be claimed before it allows key-value writes.
$ quark-cli set hello.avax/foo "hello world" --private-key-file=.different-key
<<COMMENT
error
COMMENT

# The prefix can be claimed if and only if
# the previous prefix (owner) info has not been expired.
# Even if the prefix is claimed by the same owner,
# all underlying key-values are deleted.
$ quark-cli claim hello.avax
<<COMMENT
success
COMMENT

`,
	RunE: setFunc,
}

// TODO: move all this to a separate client code
func setFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	pfx, key, val := getSetOp(args)
	cli := client.New(uri, requestTimeout)

	utx := &chain.SetTx{
		BaseTx: &chain.BaseTx{
			Pfx: pfx,
		},
		Key:   key,
		Value: val,
	}

	opts := []client.OpOption{client.WithPollTx(), client.WithPrefixInfo(pfx)}
	_, err = client.SignIssueTx(context.Background(), cli, utx, priv, opts...)
	return err
}

func getSetOp(args []string) (pfx []byte, key []byte, val []byte) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "expected exactly 2 arguments, got %d", len(args))
		os.Exit(128)
	}

	// [prefix/key] == "foo/bar"
	pfxKey := args[0]

	var err error
	pfx, key, _, err = parser.ParsePrefixKey(
		[]byte(pfxKey),
		parser.WithCheckPrefix(),
		parser.WithCheckKey(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse prefix %v", err)
		os.Exit(128)
	}

	val = []byte(args[1])

	return pfx, key, val
}
