// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"fmt"
	"os"

	"github.com/ava-labs/avalanchego/vms/rpcchainvm"
	"github.com/hashicorp/go-plugin"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/quarkvm/version"
	"github.com/ava-labs/quarkvm/vm"
)

func init() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StreamHandler(os.Stderr, log.LogfmtFormat())))
}

func main() {
	printVersion, err := PrintVersion()
	if err != nil {
		fmt.Printf("couldn't get config: %s", err)
		os.Exit(1)
	}
	// Print VM ID and exit
	if printVersion {
		fmt.Printf("%s@%s\n", vm.Name, version.Version)
		os.Exit(0)
	}

	// TODO: serve separate endpoint for range query
	// e.g., GET http://localhost/vm/foo returns "bar"
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: rpcchainvm.Handshake,
		Plugins: map[string]plugin.Plugin{
			"vm": rpcchainvm.New(&vm.VM{}),
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
