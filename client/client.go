// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package client implements "quarkvm" client SDK.
package client

import (
	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/vm"
)

// GetPrefixInfo returns the prefix information for the given prefix.
func GetPrefixInfo(requester rpc.EndpointRequester, prefix []byte) (*chain.PrefixInfo, error) {
	resp := new(vm.PrefixInfoReply)
	if err := requester.SendRequest(
		"prefixInfo",
		&vm.PrefixInfoArgs{Prefix: prefix},
		resp,
	); err != nil {
		return nil, err
	}
	return resp.Info, nil
}
