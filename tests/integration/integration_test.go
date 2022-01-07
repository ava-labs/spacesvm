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
	"github.com/ava-labs/avalanchego/utils/crypto"
	avago_version "github.com/ava-labs/avalanchego/version"
	"github.com/fatih/color"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/client"
	"github.com/ava-labs/quarkvm/vm"
)

var f *crypto.FactorySECP256K1R

func init() {
	f = &crypto.FactorySECP256K1R{}
}

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
		"timeout for transaction issuance and confirmation",
	)
	flag.IntVar(
		&vms,
		"vms",
		3,
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
	priv   crypto.PrivateKey
	sender [crypto.SECP256K1RPKLen]byte

	// when used with embedded VMs
	genesisBytes []byte
	instances    []instance
)

type instance struct {
	nodeID     ids.ShortID
	vm         *vm.VM
	toEngine   chan common.Message
	httpServer *httptest.Server
	cli        client.Client // clients for embedded VMs
}

var _ = ginkgo.BeforeSuite(func() {
	gomega.Ω(vms).Should(gomega.BeNumerically(">", 1))

	var err error
	priv, err = f.NewPrivateKey()
	gomega.Ω(err).Should(gomega.BeNil())
	sender, err = chain.FormatPK(priv.PublicKey())
	gomega.Ω(err).Should(gomega.BeNil())

	// create embedded VMs
	instances = make([]instance, vms)

	blk := &chain.StatefulBlock{
		Tmstmp:     time.Now().Unix(),
		Difficulty: minDifficulty,
		Cost:       minBlockCost,
	}
	genesisBytes, err = chain.Marshal(blk)
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

		// never trigger periodic batch gossip/block builds
		// to make testing more deterministic
		v.SetWorkInterval(24 * time.Hour)

		var hd map[string]*common.HTTPHandler
		hd, err = v.CreateHandlers()
		gomega.Ω(err).Should(gomega.BeNil())

		httpServer := httptest.NewServer(hd[""].Handler)
		instances[i] = instance{
			nodeID:     ctx.NodeID,
			vm:         v,
			toEngine:   toEngine,
			httpServer: httpServer,
			cli:        client.New(httpServer.URL, "", requestTimeout),
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
	ginkgo.It("get currently preferred block ID", func() {
		for _, inst := range instances {
			cli := inst.cli
			_, err := cli.Preferred()
			gomega.Ω(err).Should(gomega.BeNil())
		}
	})

	ginkgo.It("Gossip ClaimTx to a different node", func() {
		pfx := []byte(fmt.Sprintf("%10d", time.Now().UnixNano()))
		claimTx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{
				Sender: sender,
				Prefix: pfx,
			},
		}

		ginkgo.By("mine and issue ClaimTx", func() {
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.MineSignIssueTx(ctx, instances[0].cli, claimTx, priv)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("send gossip from node 0 to 1", func() {
			err := instances[0].vm.GossipTxs(false)
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("receive gossip in the node 1, and signal block build", func() {
			instances[1].vm.NotifyBlockReady()
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
				Sender: sender,
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

		_, err = instances[0].cli.IssueTx(tx.Bytes())
		gomega.Ω(err.Error()).Should(gomega.Equal(chain.ErrInvalidBlockID.Error()))
	})

	var pfx []byte
	ginkgo.It("Claim/SetTx with valid PoW in a single node", func() {
		pfx = []byte(fmt.Sprintf("%10d", time.Now().UnixNano()))
		claimTx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{
				Sender: sender,
				Prefix: pfx,
			},
		}

		ginkgo.By("mine and accept block with the first ClaimTx", func() {
			mineAndExpectBlkAccept(instances[0].cli, instances[0].vm, claimTx, instances[0].toEngine)
		})

		ginkgo.By("check prefix after ClaimTx has been accepted", func() {
			pf, err := instances[0].cli.PrefixInfo(pfx)
			gomega.Ω(err).To(gomega.BeNil())
			gomega.Ω(pf.Keys).To(gomega.Equal(int64(1)))
			gomega.Ω(pf.Owner).To(gomega.Equal(sender))
		})

		k, v := []byte("avax.kvm"), []byte("hello")
		setTx := &chain.SetTx{
			BaseTx: &chain.BaseTx{
				Sender: sender,
				Prefix: pfx,
			},
			Key:   k,
			Value: v,
		}

		// to work around "ErrInsufficientSurplus" for mining too fast
		time.Sleep(5 * time.Second)

		ginkgo.By("mine and accept block with a new SetTx", func() {
			mineAndExpectBlkAccept(instances[0].cli, instances[0].vm, setTx, instances[0].toEngine)
		})

		ginkgo.By("read back from VM with range query", func() {
			kvs, err := instances[0].cli.Range(pfx, k)
			gomega.Ω(err).To(gomega.BeNil())
			gomega.Ω(kvs[0].Key).To(gomega.Equal(k))
			gomega.Ω(kvs[0].Value).To(gomega.Equal(v))
		})
	})

	ginkgo.It("issue LifelineTx", func() {
		lifelineTx := &chain.LifelineTx{
			BaseTx: &chain.BaseTx{
				Sender: priv.PublicKey().Bytes(),
				Prefix: pfx,
			},
		}
		ginkgo.By("mine and accept block with a new LifelineTx", func() {
			mineAndExpectBlkAccept(instances[0].cli, instances[0].vm, lifelineTx, instances[0].toEngine)
		})
	})

	ginkgo.It("fail Set/LifelineTx with invalid key", func() {
		priv2, err := crypto.NewPrivateKey()
		gomega.Ω(err).Should(gomega.BeNil())

		ginkgo.By("SetTx fails with invalid key", func() {
			k, v := []byte("avax.kvm"), []byte("hello")
			setTx := &chain.SetTx{
				BaseTx: &chain.BaseTx{
					Sender: priv.PublicKey().Bytes(),
					Prefix: pfx,
				},
				Key:   k,
				Value: v,
			}
			expectIssueTxError(
				instances[0].cli,
				setTx,
				priv2,
				chain.ErrInvalidSignature,
			)
		})

		ginkgo.By("LifelineTx fails with invalid key", func() {
			lifelineTx := &chain.LifelineTx{
				BaseTx: &chain.BaseTx{
					Sender: priv.PublicKey().Bytes(),
					Prefix: pfx,
				},
			}
			expectIssueTxError(
				instances[0].cli,
				lifelineTx,
				priv2,
				chain.ErrInvalidSignature,
			)
		})
	})

	ginkgo.It("fail Gossip ClaimTx to a stale node when missing previous blocks", func() {
		pfx := []byte(fmt.Sprintf("%10d", time.Now().UnixNano()))
		claimTx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{
				Sender: sender,
				Prefix: pfx,
			},
		}

		ginkgo.By("mine and issue ClaimTx", func() {
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.MineSignIssueTx(ctx, instances[0].cli, claimTx, priv)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		// since the block from previous test spec has not been replicated yet
		ginkgo.By("send gossip from node 0 to 1 should fail on server-side since 1 doesn't have the block yet", func() {
			err := instances[0].vm.GossipTxs(false)
			gomega.Ω(err).Should(gomega.BeNil())

			// mempool in 1 should be empty, since gossip/submit failed
			gomega.Ω(instances[1].vm.Mempool().Len()).Should(gomega.Equal(0))
		})
	})

	// TODO: full replicate blocks between nodes
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

	// manually signal ready
	vm.NotifyBlockReady()
	// manually ack ready sig as in engine
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

func expectIssueTxError(
	cli client.Client,
	utx chain.UnsignedTransaction,
	priv *crypto.PrivateKey,
	expectedErr error,
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
	gomega.Ω(err.Error()).Should(gomega.Equal(expectedErr.Error()))
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
