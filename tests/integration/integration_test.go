// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// integration implements the integration tests.
package integration_test

import (
	"context"
	"flag"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/database/manager"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	avago_version "github.com/ava-labs/avalanchego/version"
	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/client"
	"github.com/ava-labs/quarkvm/crypto"
	"github.com/ava-labs/quarkvm/parser"
	"github.com/ava-labs/quarkvm/vm"
	"github.com/fatih/color"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestIntegration(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "quarkvm integration test suites")
}

var (
	requestTimeout time.Duration
	vms            int
	minDifficulty  uint64
	minBlockCost   uint64
)

func init() {
	flag.DurationVar(
		&requestTimeout,
		"request-timeout",
		30*time.Second,
		"set it to 0 to not wait for transaction confirmation",
	)
	flag.IntVar(
		&vms,
		"vms",
		1,
		"number of VMs to create",
	)
	flag.Uint64Var(
		&minDifficulty,
		"min-difficulty",
		chain.MinDifficulty,
		"minimum difficulty for mining",
	)
	flag.Uint64Var(
		&minBlockCost,
		"min-block-cost",
		chain.MinBlockCost,
		"minimum block cost",
	)
}

var (
	priv *crypto.PrivateKey

	// clients for embedded VMs
	// TODO: test against external endpoints
	clients []client.Client

	// when used with embedded VMs
	genesisBytes []byte
	instances    []instance
	toEngines    []chan common.Message
)

type instance struct {
	vm         *vm.VM
	httpServer *httptest.Server
}

var _ = ginkgo.BeforeSuite(func() {
	var err error
	priv, err = crypto.NewPrivateKey()
	gomega.Ω(err).Should(gomega.BeNil())

	// create embedded VMs
	clients = make([]client.Client, vms)
	instances = make([]instance, vms)
	toEngines = make([]chan common.Message, vms)

	blk := &chain.StatefulBlock{
		Tmstmp:     time.Now().Unix(),
		Difficulty: minDifficulty,
		Cost:       minBlockCost,
	}
	genesisBytes, err = chain.Marshal(blk)
	gomega.Ω(err).Should(gomega.BeNil())

	ctx := &snow.Context{
		NetworkID: 1,
		SubnetID:  ids.GenerateTestID(),
		ChainID:   ids.GenerateTestID(),
		NodeID:    ids.ShortID{1, 2, 3},
	}

	for i := range instances {
		db := manager.NewMemDB(avago_version.CurrentDatabase)
		toEngine := make(chan common.Message, 1)

		v := &vm.VM{}
		err := v.Initialize(
			ctx,
			db,
			genesisBytes,
			nil,
			nil,
			toEngine,
			nil,
			nil,
		)
		gomega.Ω(err).Should(gomega.BeNil())

		var hd map[string]*common.HTTPHandler
		hd, err = v.CreateHandlers()
		gomega.Ω(err).Should(gomega.BeNil())

		httpServer := httptest.NewServer(hd[""].Handler)
		instances[i] = instance{vm: v, httpServer: httpServer}
		clients[i] = client.New(httpServer.URL, "", requestTimeout)
		toEngines[i] = toEngine
	}

	color.Blue("created %d VMs", vms)
})

var _ = ginkgo.AfterSuite(func() {
	for _, iv := range instances {
		iv.httpServer.Close()
		err := iv.vm.Shutdown()
		gomega.Ω(err).Should(gomega.BeNil())
	}
})

var _ = ginkgo.Describe("[Ping]", func() {
	ginkgo.It("can ping", func() {
		for _, cli := range clients {
			ok, err := cli.Ping()
			gomega.Ω(ok).Should(gomega.BeTrue())
			gomega.Ω(err).Should(gomega.BeNil())
		}
	})
})

var _ = ginkgo.Describe("[ClaimTx]", func() {
	ginkgo.It("get currently preferred block ID", func() {
		for _, cli := range clients {
			_, err := cli.Preferred()
			gomega.Ω(err).Should(gomega.BeNil())
		}
	})

	ginkgo.It("fail ClaimTx with no block ID", func() {
		utx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{
				Sender: priv.PublicKey().Bytes(),
				Prefix: []byte("foo"),
			},
		}

		b, err := chain.UnsignedBytes(utx)
		gomega.Ω(err).Should(gomega.BeNil())

		sig, err := priv.Sign(b)
		gomega.Ω(err).Should(gomega.BeNil())

		tx := chain.NewTx(utx, sig)
		err = tx.Init()
		gomega.Ω(err).Should(gomega.BeNil())

		_, err = clients[0].IssueTx(tx.Bytes())
		gomega.Ω(err.Error()).Should(gomega.Equal(chain.ErrInvalidBlockID.Error()))
	})

	ginkgo.It("ClaimTx with valid PoW", func() {
		pfx := []byte(fmt.Sprintf("%10d", ginkgo.GinkgoRandomSeed()))
		claimTx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{
				Sender: priv.PublicKey().Bytes(),
				Prefix: pfx,
			},
		}

		ginkgo.By("mine and accept block with the first ClaimTx", func() {
			mineAndExpectBlkAccept(clients[0], instances[0].vm, claimTx, toEngines[0])
		})

		ginkgo.By("check prefix after ClaimTx has been accepted", func() {
			pf, err := clients[0].PrefixInfo(pfx)
			gomega.Ω(err).To(gomega.BeNil())
			gomega.Ω(pf.Keys).To(gomega.Equal(int64(1)))
			gomega.Ω(pf.Owner).To(gomega.Equal(priv.PublicKey().Bytes()))
		})

		k, v := []byte("avax.kvm"), []byte("hello")
		setTx := &chain.SetTx{
			BaseTx: &chain.BaseTx{
				Sender: priv.PublicKey().Bytes(),
				Prefix: pfx,
			},
			Key:   k,
			Value: v,
		}

		// to work around "ErrInsufficientSurplus" for mining too fast
		time.Sleep(5 * time.Second)

		ginkgo.By("mine and accept block with a new SetTx", func() {
			mineAndExpectBlkAccept(clients[0], instances[0].vm, setTx, toEngines[0])
		})

		ginkgo.By("read back from VM with range query", func() {
			kvs, err := clients[0].Range(pfx, k)
			gomega.Ω(err).To(gomega.BeNil())
			gomega.Ω(kvs[0].Key).To(gomega.Equal(append(append(pfx, parser.Delimiter), k...)))
			gomega.Ω(kvs[0].Value).To(gomega.Equal(v))
		})
	})
})

func mineAndExpectBlkAccept(
	cli client.Client,
	vm *vm.VM,
	utx chain.UnsignedTransaction,
	toEngine <-chan common.Message,
) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	mtx, err := cli.Mine(ctx, utx)
	cancel()
	gomega.Ω(err).Should(gomega.BeNil())

	b, err := chain.UnsignedBytes(mtx)
	gomega.Ω(err).Should(gomega.BeNil())

	sig, err := priv.Sign(b)
	gomega.Ω(err).Should(gomega.BeNil())

	tx := chain.NewTx(mtx, sig)
	err = tx.Init()
	gomega.Ω(err).To(gomega.BeNil())

	// or to use VM directly
	// err = vm.Submit(tx)
	_, err = cli.IssueTx(tx.Bytes())
	gomega.Ω(err).To(gomega.BeNil())

	<-toEngine

	blk, err := vm.BuildBlock()
	gomega.Ω(err).To(gomega.BeNil())

	gomega.Ω(blk.Verify()).To(gomega.BeNil())
	gomega.Ω(blk.Status()).To(gomega.Equal(choices.Processing))

	err = vm.SetPreference(blk.ID())
	gomega.Ω(err).To(gomega.BeNil())

	gomega.Ω(blk.Accept()).To(gomega.BeNil())
	gomega.Ω(blk.Status()).To(gomega.Equal(choices.Accepted))

	lastAccepted, err := vm.LastAccepted()
	gomega.Ω(err).To(gomega.BeNil())
	gomega.Ω(lastAccepted).To(gomega.Equal(blk.ID()))
}

// TODO: test with multiple VMs
