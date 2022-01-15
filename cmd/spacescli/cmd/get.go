// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/parser"
)

var getCmd = &cobra.Command{
	Use:   "get [options] space/key",
	Short: "Reads a value at space/key or space/* if a * is provided",
	RunE:  getFunc,
}

// TODO: move all this to a separate client code
func getFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "expected exactly 1 argument, got %d", len(args))
		os.Exit(128)
	}
	cli := client.New(uri, requestTimeout)
	p := args[0]
	splits := strings.Split(p, parser.Delimiter)
	kvs := []*chain.KeyValue{}
	if len(splits) == 2 && parser.CheckContents(splits[0]) == nil && splits[1] == "*" {
		_, values, err := cli.Info(splits[0])
		if err != nil {
			return err
		}
		kvs = append(kvs, values...)
	} else {
		_, v, err := cli.Resolve(args[0])
		if err != nil {
			return err
		}
		kvs = append(kvs, &chain.KeyValue{
			Key:   splits[1],
			Value: v,
		})
	}

	for _, kv := range kvs {
		fmt.Printf("%s=>%q\n", kv.Key, kv.Value)
	}
	return nil
}
