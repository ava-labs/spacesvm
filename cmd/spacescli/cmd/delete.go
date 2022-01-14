// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
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

It assumes the prefix is already claimed via "spaces-cli claim",
and the key already exists via "spaces-cli set". Otherwise, the
transaction will fail.

# claims the prefix "hello.avax"
# "hello.avax" is the prefix (or namespace)
$ spaces-cli claim hello.avax
<<COMMENT
success
COMMENT

# writes a key-value pair for the given namespace (prefix)
# by issuing "SetTx" preceded by "IssueTx" on the prefix:
# "hello.avax" is the prefix (or namespace)
# "foo" is the key
# "hello world" is the value
$ spaces-cli set hello.avax/foo "hello world"
<<COMMENT
success
COMMENT

# The prefix and key can be deleted by "delete" command.
$ spaces-cli delete hello.avax/foo
<<COMMENT
success
COMMENT

# The prefix itself cannot be deleted by "delete" command.
$ spaces-cli delete hello.avax
<<COMMENT
error
COMMENT

# The existing key-value cannot be overwritten by a different owner.
# The prefix must be claimed before it allows key-value writes.
$ spaces-cli set hello.avax/foo "hello world" --private-key-file=.different-key
<<COMMENT
error
COMMENT

`,
	RunE: deleteFunc,
}

// TODO: move all this to a separate client code
func deleteFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	space, key := getPathOp(args)
	cli := client.New(uri, requestTimeout)

	utx := &chain.DeleteTx{
		BaseTx: &chain.BaseTx{},
		Space:  space,
		Key:    key,
	}

	_, err = client.SignIssueTx(context.Background(), cli, utx, priv, space)
	return err
}
