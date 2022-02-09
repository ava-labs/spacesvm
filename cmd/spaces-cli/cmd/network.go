// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/spacesvm/client"
)

var networkCmd = &cobra.Command{
	Use:   "network [options]",
	Short: "View information about this instance of the SpacesVM",
	RunE:  networkFunc,
}

func networkFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("expected exactly 0 arguments, got %d", len(args))
	}
	cli := client.New(uri, requestTimeout)
	networkID, subnetID, chainID, err := cli.Network(context.Background())
	if err != nil {
		return err
	}
	color.Cyan("networkID=%d subnetID=%s chainID=%s", networkID, subnetID, chainID)
	return nil
}
