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

var infoCmd = &cobra.Command{
	Use:   "info [options] space",
	Short: "Reads space info and all values at space",
	RunE:  infoFunc,
}

func infoFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected exactly 1 argument, got %d", len(args))
	}
	cli := client.New(uri, requestTimeout)
	info, values, err := cli.Info(context.Background(), args[0])
	if err != nil {
		return err
	}

	client.PPInfo(info)
	for _, kv := range values {
		hr, err := json.Marshal(kv.ValueMeta)
		if err != nil {
			return err
		}
		color.Yellow("%s=>%s", kv.Key, string(hr))
	}
	return nil
}
