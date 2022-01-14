// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// integration implements the integration tests.
package integration_test

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"flag"
	"math/rand"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/database/manager"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	avago_version "github.com/ava-labs/avalanchego/version"
	ecommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	log "github.com/inconshreveable/log15"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/client"
	"github.com/ava-labs/quarkvm/parser"
	"github.com/ava-labs/quarkvm/vm"
)

func TestIntegration(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "quarkvm integration test suites")
}

var (
	requestTimeout time.Duration
	vms            int
	minPrice       int64
	minBlockCost   int64
)

func init() {
	flag.DurationVar(
		&requestTimeout,
		"request-timeout",
		120*time.Second,
		"timeout for transaction issuance and confirmation",
	)
	flag.IntVar(
		&vms,
		"vms",
		3,
		"number of VMs to create",
	)
	flag.Int64Var(
		&minPrice,
		"min-price",
		-1,
		"minimum price",
	)
	flag.Int64Var(
		&minBlockCost,
		"min-block-cost",
		-1,
		"minimum block cost",
	)
}

var (
	priv   *ecdsa.PrivateKey
	sender ecommon.Address

	// when used with embedded VMs
	genesisBytes []byte
	instances    []instance

	genesis *chain.Genesis
)

type instance struct {
	nodeID     ids.ShortID
	vm         *vm.VM
	toEngine   chan common.Message
	httpServer *httptest.Server
	cli        client.Client // clients for embedded VMs
	builder    *vm.ManualBuilder
}

var _ = ginkgo.BeforeSuite(func() {
	gomega.Ω(vms).Should(gomega.BeNumerically(">", 1))

	var err error
	priv, err = crypto.GenerateKey()
	gomega.Ω(err).Should(gomega.BeNil())
	sender = crypto.PubkeyToAddress(priv.PublicKey)

	log.Debug("generated key", "addr", sender, "priv", hex.EncodeToString(crypto.FromECDSA(priv)))

	// create embedded VMs
	instances = make([]instance, vms)

	genesis = chain.DefaultGenesis()
	if minPrice >= 0 {
		genesis.MinPrice = uint64(minPrice)
	}
	if minBlockCost >= 0 {
		genesis.MinBlockCost = uint64(minBlockCost)
	}
	genesis.Magic = rand.Uint64()
	genesis.Allocations = []*chain.Allocation{
		{
			Address: sender.Hex(),
			Balance: 10000000,
		},
	}
	genesisBytes, err = json.Marshal(genesis)
	gomega.Ω(err).Should(gomega.BeNil())

	networkID := uint32(1)
	subnetID := ids.GenerateTestID()
	chainID := ids.GenerateTestID()

	app := &appSender{}
	for i := range instances {
		ctx := &snow.Context{
			NetworkID: networkID,
			SubnetID:  subnetID,
			ChainID:   chainID,
			NodeID:    ids.GenerateTestShortID(),
		}

		toEngine := make(chan common.Message, 1)
		db := manager.NewMemDB(avago_version.CurrentDatabase)

		// TODO: test appsender
		v := &vm.VM{}
		err := v.Initialize(
			ctx,
			db,
			genesisBytes,
			nil,
			nil,
			toEngine,
			nil,
			app,
		)
		gomega.Ω(err).Should(gomega.BeNil())

		var mb *vm.ManualBuilder
		v.SetBlockBuilder(func() vm.BlockBuilder {
			mb = v.NewManualBuilder()
			return mb
		})

		var hd map[string]*common.HTTPHandler
		hd, err = v.CreateHandlers()
		gomega.Ω(err).Should(gomega.BeNil())

		httpServer := httptest.NewServer(hd[vm.PublicEndpoint].Handler)
		instances[i] = instance{
			nodeID:     ctx.NodeID,
			vm:         v,
			toEngine:   toEngine,
			httpServer: httpServer,
			cli:        client.New(httpServer.URL, requestTimeout),
			builder:    mb,
		}
	}

	// Verify genesis allocations loaded correctly (do here otherwise test may
	// check during and it will be inaccurate)
	for _, inst := range instances {
		cli := inst.cli
		g, err := cli.Genesis()
		gomega.Ω(err).Should(gomega.BeNil())

		for _, alloc := range g.Allocations {
			paddr := ecommon.HexToAddress(alloc.Address)
			bal, err := cli.Balance(paddr)
			gomega.Ω(err).Should(gomega.BeNil())
			gomega.Ω(bal).Should(gomega.Equal(alloc.Balance))
		}
	}

	app.instances = instances
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
		for _, inst := range instances {
			cli := inst.cli
			ok, err := cli.Ping()
			gomega.Ω(ok).Should(gomega.BeTrue())
			gomega.Ω(err).Should(gomega.BeNil())
		}
	})
})

var _ = ginkgo.Describe("[ClaimTx]", func() {
	ginkgo.It("get currently accepted block ID", func() {
		for _, inst := range instances {
			cli := inst.cli
			_, err := cli.Accepted()
			gomega.Ω(err).Should(gomega.BeNil())
		}
	})

	ginkgo.It("Gossip ClaimTx to a different node", func() {
		pfx := []byte(strings.Repeat("a", parser.MaxPrefixSize))
		claimTx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{
				Pfx: pfx,
			},
		}

		ginkgo.By("mine and issue ClaimTx", func() {
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.SignIssueTx(ctx, instances[0].cli, claimTx, priv)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("send gossip from node 0 to 1", func() {
			newTxs := instances[0].vm.Mempool().NewTxs(genesis.TargetUnits)
			gomega.Ω(len(newTxs)).To(gomega.Equal(1))

			err := instances[0].vm.Network().GossipNewTxs(newTxs)
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("receive gossip in the node 1, and signal block build", func() {
			instances[1].builder.NotifyBuild()
			<-instances[1].toEngine
		})

		ginkgo.By("build block in the node 1", func() {
			blk, err := instances[1].vm.BuildBlock()
			gomega.Ω(err).To(gomega.BeNil())

			gomega.Ω(blk.Verify()).To(gomega.BeNil())
			gomega.Ω(blk.Status()).To(gomega.Equal(choices.Processing))

			err = instances[1].vm.SetPreference(blk.ID())
			gomega.Ω(err).To(gomega.BeNil())

			gomega.Ω(blk.Accept()).To(gomega.BeNil())
			gomega.Ω(blk.Status()).To(gomega.Equal(choices.Accepted))

			lastAccepted, err := instances[1].vm.LastAccepted()
			gomega.Ω(err).To(gomega.BeNil())
			gomega.Ω(lastAccepted).To(gomega.Equal(blk.ID()))
		})
	})

	ginkgo.It("fail ClaimTx with no block ID", func() {
		utx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{
				Pfx: []byte("foo"),
			},
		}

		dh := chain.DigestHash(utx)
		sig, err := crypto.Sign(dh, priv)
		gomega.Ω(err).Should(gomega.BeNil())

		tx := chain.NewTx(utx, sig)
		err = tx.Init(genesis)
		gomega.Ω(err).Should(gomega.BeNil())

		_, err = instances[0].cli.IssueTx(tx.Bytes())
		gomega.Ω(err.Error()).Should(gomega.Equal(chain.ErrInvalidBlockID.Error()))
	})

	ginkgo.It("Claim/SetTx in a single node", func() {
		pfx := []byte(strings.Repeat("b", parser.MaxPrefixSize))
		claimTx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{
				Pfx: pfx,
			},
		}

		ginkgo.By("mine and accept block with the first ClaimTx", func() {
			bpfx := []byte("junk")
			instances[0].vm.SetBeneficiary(bpfx)

			blk := expectBlkAccept(instances[0], claimTx)
			gomega.Ω(blk.Beneficiary).Should(gomega.BeEmpty())

			instances[0].vm.SetBeneficiary(nil)
		})

		ginkgo.By("check prefix after ClaimTx has been accepted", func() {
			pf, err := instances[0].cli.PrefixInfo(pfx)
			gomega.Ω(err).To(gomega.BeNil())
			gomega.Ω(pf).NotTo(gomega.BeNil())
			gomega.Ω(pf.Units).To(gomega.Equal(uint64(1)))
			gomega.Ω(pf.Owner).To(gomega.Equal(sender))
		})

		k, v := []byte("avax.kvm"), []byte("hello")
		setTx := &chain.SetTx{
			BaseTx: &chain.BaseTx{
				Pfx: pfx,
			},
			Key:   k,
			Value: v,
		}

		// to work around "ErrInsufficientSurplus" for mining too fast
		time.Sleep(5 * time.Second)

		ginkgo.By("mine and accept block with a new SetTx (with beneficiary)", func() {
			i, err := instances[0].cli.PrefixInfo(pfx)
			gomega.Ω(err).To(gomega.BeNil())
			instances[0].vm.SetBeneficiary(pfx)

			blk := expectBlkAccept(instances[0], setTx)
			gomega.Ω(blk.Beneficiary).Should(gomega.Equal(pfx))

			i2, err := instances[0].cli.PrefixInfo(pfx)
			gomega.Ω(err).To(gomega.BeNil())
			n := uint64(time.Now().Unix())
			irem := (i.Expiry - n) * i.Units
			i2rem := (i2.Expiry - n) * i2.Units
			gomega.Ω(i2rem > irem).To(gomega.BeTrue())

			instances[0].vm.SetBeneficiary(nil)
		})

		ginkgo.By("read back from VM with range query", func() {
			kvs, err := instances[0].cli.Range(pfx, k)
			gomega.Ω(err).To(gomega.BeNil())
			gomega.Ω(kvs[0].Key).To(gomega.Equal(k))
			gomega.Ω(kvs[0].Value).To(gomega.Equal(v))
		})

		ginkgo.By("read back from VM with resolve", func() {
			exists, value, err := instances[0].cli.Resolve(string(pfx) + "/" + string(k))
			gomega.Ω(err).To(gomega.BeNil())
			gomega.Ω(exists).To(gomega.BeTrue())
			gomega.Ω(value).To(gomega.Equal(v))
		})
	})

	ginkgo.It("fail Gossip ClaimTx to a stale node when missing previous blocks", func() {
		pfx := []byte(strings.Repeat("c", parser.MaxPrefixSize))
		claimTx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{
				Pfx: pfx,
			},
		}

		ginkgo.By("mine and issue ClaimTx", func() {
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.SignIssueTx(ctx, instances[0].cli, claimTx, priv)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		// since the block from previous test spec has not been replicated yet
		ginkgo.By("send gossip from node 0 to 1 should fail on server-side since 1 doesn't have the block yet", func() {
			newTxs := instances[0].vm.Mempool().NewTxs(genesis.TargetUnits)
			gomega.Ω(len(newTxs)).To(gomega.Equal(1))

			err := instances[0].vm.Network().GossipNewTxs(newTxs)
			gomega.Ω(err).Should(gomega.BeNil())

			// mempool in 1 should be empty, since gossip/submit failed
			gomega.Ω(instances[1].vm.Mempool().Len()).Should(gomega.Equal(0))
		})
	})

	// TODO: full replicate blocks between nodes
})

func expectBlkAccept(
	i instance,
	utx chain.UnsignedTransaction,
) *chain.StatelessBlock {
	g, err := i.cli.Genesis()
	gomega.Ω(err).Should(gomega.BeNil())
	utx.SetMagic(g.Magic)

	la, err := i.cli.Accepted()
	gomega.Ω(err).Should(gomega.BeNil())
	utx.SetBlockID(la)

	price, blockCost, err := i.cli.SuggestedFee()
	gomega.Ω(err).Should(gomega.BeNil())
	utx.SetPrice(price + blockCost/utx.FeeUnits(g))

	sig, err := crypto.Sign(chain.DigestHash(utx), priv)
	gomega.Ω(err).Should(gomega.BeNil())

	tx := chain.NewTx(utx, sig)
	err = tx.Init(genesis)
	gomega.Ω(err).To(gomega.BeNil())

	// or to use VM directly
	// err = vm.Submit(tx)
	_, err = i.cli.IssueTx(tx.Bytes())
	gomega.Ω(err).To(gomega.BeNil())

	// manually signal ready
	i.builder.NotifyBuild()
	// manually ack ready sig as in engine
	<-i.toEngine

	blk, err := i.vm.BuildBlock()
	gomega.Ω(err).To(gomega.BeNil())

	gomega.Ω(blk.Verify()).To(gomega.BeNil())
	gomega.Ω(blk.Status()).To(gomega.Equal(choices.Processing))

	err = i.vm.SetPreference(blk.ID())
	gomega.Ω(err).To(gomega.BeNil())

	gomega.Ω(blk.Accept()).To(gomega.BeNil())
	gomega.Ω(blk.Status()).To(gomega.Equal(choices.Accepted))

	lastAccepted, err := i.vm.LastAccepted()
	gomega.Ω(err).To(gomega.BeNil())
	gomega.Ω(lastAccepted).To(gomega.Equal(blk.ID()))

	return blk.(*chain.StatelessBlock)
}

var _ common.AppSender = &appSender{}

type appSender struct {
	next      int
	instances []instance
}

func (app *appSender) SendAppGossip(appGossipBytes []byte) error {
	n := len(app.instances)
	sender := app.instances[app.next].nodeID
	app.next++
	app.next %= n
	return app.instances[app.next].vm.AppGossip(sender, appGossipBytes)
}

func (app *appSender) SendAppRequest(_ ids.ShortSet, _ uint32, _ []byte) error { return nil }
func (app *appSender) SendAppResponse(_ ids.ShortID, _ uint32, _ []byte) error { return nil }
func (app *appSender) SendAppGossipSpecific(_ ids.ShortSet, _ []byte) error    { return nil }
