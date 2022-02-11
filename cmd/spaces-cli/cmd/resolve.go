// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/client"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve [options] space/key",
	Short: "Reads a value at space/key",
	RunE:  resolveFunc,
}

func resolveFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected exactly 1 argument, got %d", len(args))
	}
	cli := client.New(uri, requestTimeout)
	_, v, vmeta, err := cli.Resolve(context.Background(), args[0])
	if err != nil {
		return err
	}

	color.Yellow("%s=>%q", args[0], v)
	hr, err := json.Marshal(vmeta)
	if err != nil {
		return err
	}
	color.Yellow("Metadata: %s", string(hr))

	color.Green("resolved %s", args[0])
	return nil
}
