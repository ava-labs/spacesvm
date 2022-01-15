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

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/parser"
	"github.com/ava-labs/spacesvm/vm"
)

func TestIntegration(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "spacesvm integration test suites")
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

	priv2   *ecdsa.PrivateKey
	sender2 ecommon.Address

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

	priv2, err = crypto.GenerateKey()
	gomega.Ω(err).Should(gomega.BeNil())
	sender2 = crypto.PubkeyToAddress(priv2.PublicKey)

	log.Debug("generated key", "addr", sender2, "priv", hex.EncodeToString(crypto.FromECDSA(priv2)))

	// create embedded VMs
	instances = make([]instance, vms)

	genesis = chain.DefaultGenesis()
	if minPrice >= 0 {
		genesis.MinPrice = uint64(minPrice)
	}
	if minBlockCost >= 0 {
		genesis.MinBlockCost = uint64(minBlockCost)
	}
	genesis.Magic = 5
	genesis.BlockTarget = 0 // disable block throttling
	genesis.Allocations = []*chain.Allocation{
		{
			Address: sender,
			Balance: 10000000,
		},
		{
			Address: sender2,
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
			bal, err := cli.Balance(alloc.Address)
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

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))] //nolint:gosec
	}
	return string(b)
}

var _ = ginkgo.Describe("[ClaimTx]", func() {
	ginkgo.It("ensure activity yet", func() {
		activity, err := instances[0].cli.RecentActivity()
		gomega.Ω(err).To(gomega.BeNil())

		gomega.Ω(len(activity)).To(gomega.Equal(0))
	})

	ginkgo.It("get currently accepted block ID", func() {
		for _, inst := range instances {
			cli := inst.cli
			_, err := cli.Accepted()
			gomega.Ω(err).Should(gomega.BeNil())
		}
	})

	ginkgo.It("Gossip ClaimTx to a different node", func() {
		space := strings.Repeat("a", parser.MaxIdentifierSize)
		claimTx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{},
			Space:  space,
		}

		ginkgo.By("mine and issue ClaimTx", func() {
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.SignIssueRawTx(ctx, instances[0].cli, claimTx, priv)
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

		ginkgo.By("ensure all activity accounted for", func() {
			activity, err := instances[1].cli.RecentActivity()
			gomega.Ω(err).To(gomega.BeNil())

			gomega.Ω(len(activity)).To(gomega.Equal(1))
			a0 := activity[0]
			gomega.Ω(a0.Typ).To(gomega.Equal("claim"))
			gomega.Ω(a0.Space).To(gomega.Equal(space))
			gomega.Ω(a0.Sender).To(gomega.Equal(sender.Hex()))
		})
	})

	ginkgo.It("fail ClaimTx with no block ID", func() {
		utx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{},
			Space:  "foo",
		}

		dh, err := chain.DigestHash(utx)
		gomega.Ω(err).Should(gomega.BeNil())
		sig, err := crypto.Sign(dh, priv)
		gomega.Ω(err).Should(gomega.BeNil())

		tx := chain.NewTx(utx, sig)
		err = tx.Init(genesis)
		gomega.Ω(err).Should(gomega.BeNil())

		_, err = instances[0].cli.IssueRawTx(tx.Bytes())
		gomega.Ω(err.Error()).Should(gomega.Equal(chain.ErrInvalidBlockID.Error()))
	})

	ginkgo.It("Claim/SetTx in a single node", func() {
		space := strings.Repeat("b", parser.MaxIdentifierSize)
		claimTx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{},
			Space:  space,
		}

		ginkgo.By("mine and accept block with the first ClaimTx", func() {
			expectBlkAccept(instances[0], claimTx, priv)
		})

		ginkgo.By("check prefix after ClaimTx has been accepted", func() {
			pf, values, err := instances[0].cli.Info(space)
			gomega.Ω(err).To(gomega.BeNil())
			gomega.Ω(pf).NotTo(gomega.BeNil())
			gomega.Ω(pf.Units).To(gomega.Equal(uint64(1)))
			gomega.Ω(pf.Owner).To(gomega.Equal(sender))
			gomega.Ω(len(values)).To(gomega.Equal(0))
		})

		k, v := "avax.kvm", []byte("hello")
		setTx := &chain.SetTx{
			BaseTx: &chain.BaseTx{},
			Space:  space,
			Key:    k,
			Value:  v,
		}

		ginkgo.By("accept block with a new SetTx", func() {
			expectBlkAccept(instances[0], setTx, priv)
		})

		ginkgo.By("read back from VM with range query", func() {
			_, kvs, err := instances[0].cli.Info(space)
			gomega.Ω(err).To(gomega.BeNil())
			gomega.Ω(kvs[0].Key).To(gomega.Equal(k))
			gomega.Ω(kvs[0].Value).To(gomega.Equal(v))
		})

		ginkgo.By("read back from VM with resolve", func() {
			exists, value, err := instances[0].cli.Resolve(space + "/" + k)
			gomega.Ω(err).To(gomega.BeNil())
			gomega.Ω(exists).To(gomega.BeTrue())
			gomega.Ω(value).To(gomega.Equal(v))
		})

		ginkgo.By("transfer funds to other sender", func() {
			transferTx := &chain.TransferTx{
				BaseTx: &chain.BaseTx{},
				To:     sender2,
				Units:  100,
			}
			expectBlkAccept(instances[0], transferTx, priv)
		})

		ginkgo.By("move space to other sender", func() {
			moveTx := &chain.MoveTx{
				BaseTx: &chain.BaseTx{},
				To:     sender2,
				Space:  space,
			}
			expectBlkAccept(instances[0], moveTx, priv)
		})

		ginkgo.By("ensure all activity accounted for", func() {
			activity, err := instances[0].cli.RecentActivity()
			gomega.Ω(err).To(gomega.BeNil())

			gomega.Ω(len(activity)).To(gomega.Equal(5))
			a0 := activity[0]
			gomega.Ω(a0.Typ).To(gomega.Equal("move"))
			gomega.Ω(a0.Space).To(gomega.Equal(space))
			gomega.Ω(a0.To).To(gomega.Equal(sender2.Hex()))
			gomega.Ω(a0.Sender).To(gomega.Equal(sender.Hex()))
			a1 := activity[1]
			gomega.Ω(a1.Typ).To(gomega.Equal("transfer"))
			gomega.Ω(a1.To).To(gomega.Equal(sender2.Hex()))
			gomega.Ω(a1.Units).To(gomega.Equal(uint64(100)))
			gomega.Ω(a1.Sender).To(gomega.Equal(sender.Hex()))
			a2 := activity[2]
			gomega.Ω(a2.Typ).To(gomega.Equal("set"))
			gomega.Ω(a2.Space).To(gomega.Equal(space))
			gomega.Ω(a2.Key).To(gomega.Equal(k))
			gomega.Ω(a2.Sender).To(gomega.Equal(sender.Hex()))
			a3 := activity[3]
			gomega.Ω(a3.Typ).To(gomega.Equal("claim"))
			gomega.Ω(a3.Space).To(gomega.Equal(space))
			gomega.Ω(a3.Sender).To(gomega.Equal(sender.Hex()))
		})
	})

	ginkgo.It("Distribute Lottery Reward", func() {
		ginkgo.By("ensure that sender is rewarded at least once", func() {
			bal, err := instances[0].cli.Balance(sender)
			gomega.Ω(err).To(gomega.BeNil())

			found := false
			for i := 0; i < 100; i++ {
				claimTx := &chain.ClaimTx{
					BaseTx: &chain.BaseTx{},
					Space:  RandStringRunes(64),
				}

				// Use a different sender so that sender is rewarded
				expectBlkAccept(instances[0], claimTx, priv2)

				bal2, err := instances[0].cli.Balance(sender)
				gomega.Ω(err).To(gomega.BeNil())

				if bal2 > bal {
					found = true
					break
				}
			}
			gomega.Ω(found).To(gomega.BeTrue())
		})
	})

	ginkgo.It("fail Gossip ClaimTx to a stale node when missing previous blocks", func() {
		space := strings.Repeat("c", parser.MaxIdentifierSize)
		claimTx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{},
			Space:  space,
		}

		ginkgo.By("mine and issue ClaimTx", func() {
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.SignIssueRawTx(ctx, instances[0].cli, claimTx, priv)
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
	signer *ecdsa.PrivateKey,
) {
	g, err := i.cli.Genesis()
	gomega.Ω(err).Should(gomega.BeNil())
	utx.SetMagic(g.Magic)

	la, err := i.cli.Accepted()
	gomega.Ω(err).Should(gomega.BeNil())
	utx.SetBlockID(la)

	price, blockCost, err := i.cli.SuggestedRawFee()
	gomega.Ω(err).Should(gomega.BeNil())
	utx.SetPrice(price + blockCost/utx.FeeUnits(g))

	dh, err := chain.DigestHash(utx)
	gomega.Ω(err).Should(gomega.BeNil())
	sig, err := crypto.Sign(dh, signer)
	gomega.Ω(err).Should(gomega.BeNil())

	tx := chain.NewTx(utx, sig)
	err = tx.Init(genesis)
	gomega.Ω(err).To(gomega.BeNil())

	// or to use VM directly
	// err = vm.Submit(tx)
	_, err = i.cli.IssueRawTx(tx.Bytes())
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
