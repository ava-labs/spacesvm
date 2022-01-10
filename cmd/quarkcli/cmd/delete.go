// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/client"
	"github.com/ava-labs/quarkvm/parser"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [options] <prefix/key>",
	Short: "Deletes a key-value pair for the given prefix",
	Long: `
Issues "SetTx" to delete key-value pair(s).

The prefix is automatically parsed with the delimiter "/".
When given a key "foo/hello", the "set" creates the transaction
with "foo" as prefix and "hello" as key. The prefix/key cannot
have more than one delimiter (e.g., "foo/hello/world" is invalid)
in order to maintain the flat key space.

It assumes the prefix is already claimed via "quark-cli claim",
and the key already exists via "quark-cli set". Otherwise, the
transaction will fail.

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

# The prefix and key can be deleted by "delete" command.
$ quark-cli delete hello.avax/foo
<<COMMENT
success
COMMENT

# The prefix itself cannot be deleted by "delete" command.
$ quark-cli delete hello.avax
<<COMMENT
error
COMMENT

# The existing key-value cannot be overwritten by a different owner.
# The prefix must be claimed before it allows key-value writes.
$ quark-cli set hello.avax/foo "hello world" --private-key-file=.different-key
<<COMMENT
error
COMMENT

`,
	RunE: deleteFunc,
}

// TODO: move all this to a separate client code
func deleteFunc(cmd *cobra.Command, args []string) error {
	priv, err := LoadPK(privateKeyFile)
	if err != nil {
		return err
	}
	pk, err := chain.FormatPK(priv.PublicKey())
	if err != nil {
		return err
	}

	pfx, key := getDeleteOp(args)
	cli := client.New(uri, requestTimeout)

	utx := &chain.SetTx{
		BaseTx: &chain.BaseTx{
			Sender: pk,
			Prefix: pfx,
		},
		Key:   key,
		Value: nil,
	}

	opts := []client.OpOption{client.WithPollTx(), client.WithPrefixInfo(pfx)}
	_, err = client.MineSignIssueTx(context.Background(), cli, utx, priv, opts...)
	return err
}

func getDeleteOp(args []string) (pfx []byte, key []byte) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "expected exactly 1 argument, got %d", len(args))
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

	return pfx, key
}
