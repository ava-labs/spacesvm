// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mempool_test

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/mempool"
)

// $ go install -v golang.org/x/perf/cmd/benchstat@latest
//
// $ go test -run=NONE -bench=BenchmarkMempoolAddPrune > old.txt
// # make changes
// $ go test -run=NONE -bench=BenchmarkMempoolAddPrune > new.txt
//
// $ benchstat old.txt new.txt
// name        old time/op  new time/op  delta
// Test...     18.8ns ± 0%  15.8ns ± 0%   ~     (p=1.000 n=1+1)
//
func BenchmarkMempoolAddPrune(b *testing.B) {
	b.StopTimer()

	priv, err := crypto.GenerateKey()
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		mp, sampleBlkIDs := createTestMempool(b, priv, 2000, 10000, 500)
		mp.Prune(sampleBlkIDs)
	}
}

func createTestMempool(
	b *testing.B,
	priv *ecdsa.PrivateKey,
	maxSize int,
	n int,
	sampleBlk int) (mp chain.Mempool, sampleBlkIDs ids.Set) {
	b.Helper()
	if sampleBlk*2 >= n {
		b.Fatalf("unexpected sampleBlk %d (expected < N/2 %d)", sampleBlk, n)
	}
	if n < 10 {
		b.Fatalf("expected at least 10 transactions, got %d", n)
	}

	// pre-create sampleBlk*2 block IDs
	blksN := sampleBlk * 2
	blks := make([]ids.ID, blksN)
	for i := range blks {
		blks[i] = ids.GenerateTestID()
	}

	b.StopTimer()
	g := chain.DefaultGenesis()
	txs := make([]*chain.Transaction, n)
	for i := 0; i < n; i++ {
		spc := make([]byte, 8)
		_, err := rand.Read(spc)
		if err != nil {
			b.Fatal(err)
		}

		tx := &chain.Transaction{
			UnsignedTransaction: &chain.ClaimTx{
				BaseTx: &chain.BaseTx{
					BlockID: blks[i%blksN],
				},
				Space: string(spc),
			},
		}
		sig, err := chain.Sign(tx.DigestHash(), priv)
		if err != nil {
			b.Fatal(err)
		}
		tx.Signature = sig
		if err := tx.Init(g); err != nil {
			b.Fatal(err)
		}

		txs[i] = tx
	}

	sampleBlkIDs = ids.NewSet(sampleBlk)

	mp = mempool.New(g, maxSize)

	b.StartTimer()
	for _, tx := range txs {
		if added := mp.Add(tx); added && sampleBlkIDs.Len() < sampleBlk {
			sampleBlkIDs.Add(tx.GetBlockID())
		}
	}
	return mp, sampleBlkIDs
}
