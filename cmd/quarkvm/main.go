// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ava-labs/avalanchego/vms/rpcchainvm"
	"github.com/ava-labs/quarkvm/version"
	"github.com/ava-labs/quarkvm/vm"
	"github.com/hashicorp/go-plugin"
	log "github.com/inconshreveable/log15"
)

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

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// When we get a SIGINT or SIGTERM, stop the network.
	signalsCh := make(chan os.Signal, 1)
	signal.Notify(signalsCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-ctx.Done():
			return
		case sig := <-signalsCh:
			fmt.Println("received OS signal:", sig)
			cancel()
		}
	}()

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StreamHandler(os.Stderr, log.JsonFormat())))
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: rpcchainvm.Handshake,
		Plugins: map[string]plugin.Plugin{
			"vm": rpcchainvm.New(&vm.VM{}),
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
