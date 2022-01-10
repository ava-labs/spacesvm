// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import (
	"context"
	"runtime"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/fatih/color"
	"golang.org/x/sync/errgroup"

	"github.com/ava-labs/quarkvm/chain"
)

const (
	durPrecision  = 10 * time.Millisecond
	etaMultiplier = 3
)

var (
	concurrency = uint64(runtime.NumCPU())
)

type miningData struct {
	blockID       ids.ID
	minDifficulty uint64
	minCost       uint64
}

// TODO: properly benchmark and optimize
func (cli *client) Mine(ctx context.Context, utx chain.UnsignedTransaction) (chain.UnsignedTransaction, error) {
	now := time.Now()
	g, gctx := errgroup.WithContext(ctx)

	// We purposely do not lock around any of these values because it makes the
	// core mining loop inefficient.
	var (
		ready     = make(chan struct{})
		md        *miningData
		agraffiti uint64 // approximate graffiti (could be set by any thread)
		solution  chain.UnsignedTransaction
	)

	// Mine for solution
	for i := uint64(0); i < concurrency; i++ {
		j := i             // i will get overwritten during loop iteration
		jutx := utx.Copy() // ensure each thread is modifying own copy of tx
		graffiti := j      // need to offset graffiti by thread
		g.Go(func() error {
			// Wait for all vars to be initialized
			select {
			case <-ready:
			case <-gctx.Done():
				return gctx.Err()
			}

			lastBlk := md.blockID
			for gctx.Err() == nil {
				cmd := md
				// Reset graffiti when block has been updated
				//
				// Note: We always want to use the newest BlockID when mining to maximize
				// the probability our transaction will get into a block before it
				// expires.
				if cmd.blockID != lastBlk {
					lastBlk = cmd.blockID
					graffiti = j
				}

				// Try new graffiti
				jutx.SetBlockID(cmd.blockID)
				jutx.SetGraffiti(graffiti)
				_, utxd, err := chain.CalcDifficulty(jutx)
				if err != nil {
					return err
				}
				if utxd >= cmd.minDifficulty &&
					(utxd-cmd.minDifficulty)*jutx.FeeUnits() >= cmd.minDifficulty*cmd.minCost {
					solution = jutx
					color.Green(
						"mining complete[%d] (difficulty=%d, surplus=%d, elapsed=%v)",
						graffiti, utxd, (utxd-cmd.minDifficulty)*solution.FeeUnits(), time.Since(now).Round(durPrecision),
					)
					return ErrSolution
				}

				// Work is insufficient, try again
				graffiti += concurrency // offset to avoid duplicate work
				agraffiti = graffiti    // approximate graffiti values
			}
			return gctx.Err()
		})
	}

	// Periodically print ETA
	g.Go(func() error {
		// Wait for all vars to be initialized
		select {
		case <-ready:
		case <-gctx.Done():
			return gctx.Err()
		}

		// Inline function so that we don't need to copy variables around and/or
		// make execution context with locks
		printETA := func() {
			// If we haven't returned yet, but have a solution, exit
			if solution != nil {
				return
			}

			// Assumes each additional unit of difficulty is ~1ms of compute
			cmd := md
			eta := time.Duration(utx.FeeUnits()*cmd.minDifficulty) * time.Millisecond
			eta = (eta / time.Duration(concurrency)) * etaMultiplier // account for threads and overestimate
			diff := time.Since(now)
			if diff > eta {
				color.Yellow(
					"mining in progress[%s/%d]... (elapsed=%v, threads=%d)",
					cmd.blockID, agraffiti, time.Since(now).Round(durPrecision), concurrency,
				)
			} else {
				eta -= diff
				color.Yellow(
					"mining in progress[%s/%d]... (elapsed=%v, est. remaining=%v, threads=%d)",
					cmd.blockID, agraffiti, time.Since(now).Round(durPrecision), eta.Round(durPrecision), concurrency,
				)
			}
		}

		t := time.NewTicker(2 * time.Second)
		printETA()
		for {
			select {
			case <-t.C:
				printETA()
			case <-gctx.Done():
				return gctx.Err()
			}
		}
	})

	// Periodically update blockID and required difficulty
	g.Go(func() error {
		t := time.NewTicker(time.Second)
		readyClosed := false
		for {
			select {
			case <-t.C:
				blkID, err := cli.Preferred()
				if err != nil {
					return err
				}
				diff, cost, err := cli.EstimateDifficulty()
				if err != nil {
					return err
				}

				md = &miningData{
					blockID:       blkID,
					minDifficulty: diff,
					minCost:       cost,
				}

				if !readyClosed {
					close(ready)
					readyClosed = true
				}
			case <-gctx.Done():
				return nil
			}
		}
	})
	err := g.Wait()
	if solution != nil {
		// If a solution was found, we don't care what the error was.
		return solution, nil
	}
	return nil, err
}
