// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/parser"
)

var lifelineCmd = &cobra.Command{
	Use:   "lifeline [options] <prefix>",
	Short: "Extends the life of a given prefix",
	RunE:  lifelineFunc,
}

// TODO: move all this to a separate client code
func lifelineFunc(cmd *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	pfx := getLifelineOp(args)
	cli := client.New(uri, requestTimeout)

	utx := &chain.LifelineTx{
		BaseTx: &chain.BaseTx{
			Pfx: pfx,
		},
	}

	opts := []client.OpOption{client.WithPollTx(), client.WithPrefixInfo(pfx)}
	_, err = client.SignIssueTx(context.Background(), cli, utx, priv, opts...)
	return err
}

func getLifelineOp(args []string) (pfx []byte) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "expected exactly 1 argument, got %d", len(args))
		os.Exit(128)
	}

	pfx = []byte(args[0])
	if bytes.HasSuffix(pfx, []byte{'/'}) {
		pfx = pfx[:len(pfx)-1]
	}

	// check here first before parsing in case "pfx" is empty
	if err := parser.CheckPrefix(pfx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to verify prefix %v", err)
		os.Exit(128)
	}
	if _, _, _, err := parser.ParsePrefixKey(pfx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse prefix %v", err)
		os.Exit(128)
	}

	return pfx
}
