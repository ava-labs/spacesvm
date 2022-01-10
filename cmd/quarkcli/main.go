// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// "quark-cli" implements quarkvm client operation interface.
package main

import (
	"fmt"
	"os"

	"github.com/ava-labs/quarkvm/cmd/quarkcli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "quark-cli failed %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
