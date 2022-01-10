// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import (
	"context"
	"sync"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/fatih/color"
	"golang.org/x/sync/errgroup"

	"github.com/ava-labs/quarkvm/chain"
)

func (cli *client) Mine(ctx context.Context, utx chain.UnsignedTransaction) (chain.UnsignedTransaction, error) {
	now := time.Now()
	ctx, cancel := context.WithCancel(ctx)
	g, gctx := errgroup.WithContext(ctx)

	var (
		ready         = make(chan struct{})
		dl            = sync.RWMutex{}
		blockID       ids.ID
		minDifficulty uint64
		minCost       uint64

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

		lastBlk := blockID
		for gctx.Err() == nil {
			dl.RLock()
			bID := blockID
			md := minDifficulty
			mc := minCost
			dl.RUnlock()

			// Reset graffiti when block has been updated
			//
			// Note: We always want to use the newest BlockID when mining to maximize
			// the probability our transaction will get into a block before it
			// expires.
			if bID != lastBlk {
				lastBlk = bID
				graffiti = 0
			}

			// Try new graffiti
			utx.SetBlockID(bID)
			utx.SetGraffiti(graffiti)
			_, utxd, err := chain.CalcDifficulty(utx)
			if err != nil {
				return err
			}
			if utxd >= md && (utxd-md)*utx.FeeUnits() >= mc*md {
				solution = utx
				color.Green(
					"mining complete[%d] (difficulty=%d, surplus=%d, t=%v)",
					graffiti, utxd, (utxd-minDifficulty)*solution.FeeUnits(), time.Since(now),
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

			dl.RLock()
			md := minDifficulty
			bID := blockID
			dl.RUnlock()

			// Assumes each additional unit of difficulty is ~1ms of compute
			eta := time.Duration(utx.FeeUnits()*md) * time.Millisecond * 3 / 2
			diff := time.Since(now)
			if diff > eta {
				eta = 0
			} else {
				eta -= diff
			}
			color.Yellow(
				"mining in progress[%s/%d]... (ETA=%v)",
				bID, graffiti, eta,
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

				dl.Lock()
				blockID = blkID
				minDifficulty = diff
				minCost = cost
				dl.Unlock()

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
