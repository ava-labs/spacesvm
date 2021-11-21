// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package agent implements KVVM agent.
package agent

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"ekyu.moe/cryptonight"
	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/crypto/ed25519"
	"github.com/ava-labs/quarkvm/transaction"
)

type Agent interface {
	Run()
}

type agent struct {
	ctx   context.Context
	chain chain.Chain

	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

func New(ctx context.Context, chain chain.Chain) Agent {
	prv, err := ed25519.NewPrivateKey()
	if err != nil {
		panic(err)
	}
	pub := prv.PublicKey()

	fmt.Println("new agent:", pub.Address())
	return &agent{
		ctx:   ctx,
		chain: chain,

		privateKey: prv,
		publicKey:  pub,
	}
}

func (a *agent) Run() {
	for a.ctx.Err() == nil {
		prefix := randString(16)
		if rand.Intn(100) < 20 {
			// claim own address key
			prefix = a.publicKey.Address()
			fmt.Println("attempting to claim address prefix", prefix)
		}
		utx := a.claim(prefix)
		a.mine(utx)
		stx := a.sign(utx)
		a.chain.Submit(stx)

		// wait for claim to be set or abandon
		confirmed := a.confirm(stx)
		if !confirmed {
			// TODO: try again with same prefix
			continue
		}
		owner, _, err := a.chain.GetPrefixInfo([]byte(prefix))
		if err != nil {
			panic(err)
		}
		fmt.Println("prefix claimed:", prefix, "expires:", owner.Expiry, "keys:", owner.Keys)
		// TODO: print out "rate of decay"
		// TODO: set 2 keys
		// TODO: delete 1 key
		// TODO: wait for key expiry
		// TODO: attempt to set key
		// TODO: add lifeline
	}
}

func (a *agent) claim(prefix string) transaction.Unsigned {
	return transaction.NewClaim(a.publicKey, []byte(prefix))
}

func (a *agent) mine(utx transaction.Unsigned) {
	for {
		cbID := a.chain.CurrentBlock().ID()
		utx.SetBlockID(cbID)
		graffiti := big.NewInt(0)
		for a.chain.ValidBlockID(cbID) {
			utx.SetGraffiti(graffiti.Bytes())
			h := cryptonight.Sum(transaction.UnsignedBytes(utx), 2)
			if cryptonight.CheckHash(h, a.chain.DifficultyEstimate()) {
				return
			}
			graffiti.Add(graffiti, big.NewInt(1))
		}
		// Get new block hash if no longer valid
	}
}

func (a *agent) sign(utx transaction.Unsigned) *transaction.Transaction {
	sig, err := a.privateKey.Sign(transaction.UnsignedBytes(utx))
	if err != nil {
		panic(err)
	}
	return transaction.New(utx, sig)
}

func (a *agent) confirm(stx *transaction.Transaction) bool {
	loops := 0
	for a.ctx.Err() == nil && a.chain.ValidBlockID(stx.Unsigned.GetBlockID()) {
		if a.chain.TxConfirmed(stx.ID()) {
			return true
		}
		time.Sleep(1 * time.Second)
		loops++

		// Resubmit if pending for a while but still valid
		if loops%5 == 0 && !a.chain.MempoolContains(stx.ID()) {
			a.chain.Submit(stx)
		}
	}
	return false
}

func randString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	l := rand.Intn(n) + 1 // ensure never 0

	s := make([]rune, l)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
