// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"fmt"

	"github.com/ava-labs/spacesvm/parser"
)

func getPathOp(args []string) (space string, key string, err error) {
	if len(args) != 1 {
		return "", "", fmt.Errorf("expected exactly 1 argument, got %d", len(args))
	}

	// [space/key] == "foo/bar"
	spaceKey := args[0]

	space, key, err = parser.ResolvePath(spaceKey)
	if err != nil {
		return "", "", fmt.Errorf("%w: failed to parse space", err)
	}

	return space, key, nil
}
