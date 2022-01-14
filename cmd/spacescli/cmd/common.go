// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"fmt"
	"os"

	"github.com/ava-labs/spacesvm/parser"
)

func getPathOp(args []string) (space string, key string) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "expected exactly 1 argument, got %d", len(args))
		os.Exit(128)
	}

	// [space/key] == "foo/bar"
	spaceKey := args[0]

	var err error
	space, key, err = parser.ResolvePath(spaceKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse prefix %v", err)
		os.Exit(128)
	}

	return space, key
}
