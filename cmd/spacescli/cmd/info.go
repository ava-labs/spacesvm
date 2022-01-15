// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/client"
)

var infoCmd = &cobra.Command{
	Use:   "info [options] space",
	Short: "Reads space info and all values at space",
	RunE:  infoFunc,
}

// TODO: move all this to a separate client code
func infoFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "expected exactly 1 argument, got %d", len(args))
		os.Exit(128)
	}
	cli := client.New(uri, requestTimeout)
	info, values, err := cli.Info(args[0])
	if err != nil {
		return err
	}

	client.PPInfo(info)
	for _, kv := range values {
		color.Yellow("%s=>%q", kv.Key, kv.Value)
	}
	return nil
}
