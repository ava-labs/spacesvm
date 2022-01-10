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
	ctx, cancel := context.WithCancel(ctx)
	g, gctx := errgroup.WithContext(ctx)

	var (
		ready = make(chan struct{})
		l     = sync.RWMutex{}

		blockID       ids.ID
		minDifficulty uint64
		minSurplus    uint64

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
		graffiti := uint64(0)

		for gctx.Err() == nil {
			l.RLock()
			// Reset graffiti when block has been updated
			//
			// Note: We always want to use the newest BlockID when mining to maximize
			// the probability our transaction will get into a block before it
			// expires.
			if blockID != lastBlk {
				lastBlk = blockID
				graffiti = 0
			}
			utx.SetBlockID(blockID)
			utx.SetGraffiti(graffiti)
			_, utxd, err := chain.CalcDifficulty(utx)
			if err != nil {
				l.RUnlock()
				return err
			}
			if utxd >= minDifficulty && (utxd-minDifficulty)*utx.FeeUnits() >= minSurplus {
				l.RUnlock()
				solution = utx
				cancel()
				return nil
			}
			graffiti++
			l.RUnlock()
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

		t := time.NewTimer(3 * time.Second)
		for {
			select {
			case <-t.C:
				l.RLock()
				// Assumes each additional unit of difficulty is ~1ms of compute
				eta := time.Duration(utx.FeeUnits()*minDifficulty) * time.Millisecond
				color.Yellow("mining in progress...(fee units=%d ETA=%v)", utx.FeeUnits(), eta)
				l.RUnlock()
			case <-gctx.Done():
				return gctx.Err()
			}
		}
	})

	// Periodically update blockID and required difficulty
	g.Go(func() error {
		t := time.NewTimer(time.Second)
		readyClosed := false
		for {
			select {
			case <-t.C:
				blkID, err := cli.Preferred()
				if err != nil {
					return err
				}
				diff, surplus, err := cli.EstimateDifficulty()
				if err != nil {
					return err
				}

				l.Lock()
				blockID = blkID
				minDifficulty = diff
				minSurplus = surplus
				l.Unlock()

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
