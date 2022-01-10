// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import (
	"context"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/fatih/color"
	"golang.org/x/sync/errgroup"

	"github.com/ava-labs/quarkvm/chain"
)

type miningData struct {
	blockID       ids.ID
	minDifficulty uint64
	minCost       uint64
}

func (cli *client) Mine(ctx context.Context, utx chain.UnsignedTransaction) (chain.UnsignedTransaction, error) {
	now := time.Now()
	ctx, cancel := context.WithCancel(ctx)
	g, gctx := errgroup.WithContext(ctx)

	// We purposely do not lock around any of these values because it makes the
	// core mining loop inefficient.
	var (
		ready    = make(chan struct{})
		md       *miningData
		graffiti uint64
		solution chain.UnsignedTransaction
	)

	// Mine for solution
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
				graffiti = 0
			}

			// Try new graffiti
			utx.SetBlockID(cmd.blockID)
			utx.SetGraffiti(graffiti)
			_, utxd, err := chain.CalcDifficulty(utx)
			if err != nil {
				return err
			}
			if utxd >= cmd.minDifficulty &&
				(utxd-cmd.minDifficulty)*utx.FeeUnits() >= cmd.minDifficulty*cmd.minCost {
				solution = utx
				color.Green(
					"mining complete[%d] (difficulty=%d, surplus=%d, t=%v)",
					graffiti, utxd, (utxd-cmd.minDifficulty)*solution.FeeUnits(), time.Since(now),
				)
				cancel()
				return nil
			}

			// Work is insufficient, try again
			graffiti++
		}
		return gctx.Err()
	})

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
			eta := time.Duration(utx.FeeUnits()*cmd.minDifficulty) * time.Millisecond * 2 // overestimate by 2
			diff := time.Since(now)
			if diff > eta {
				eta = 0
			} else {
				eta -= diff
			}
			color.Yellow(
				"mining in progress[%s/%d]... (ETA=%v)",
				cmd.blockID, graffiti, eta,
			)
		}

		t := time.NewTicker(3 * time.Second)
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
