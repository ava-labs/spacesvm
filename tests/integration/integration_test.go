// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// integration implements the integration tests.
package integration_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/database/manager"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/utils/units"
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
	"github.com/ava-labs/spacesvm/tdata"
	"github.com/ava-labs/spacesvm/tree"
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
	genesis.Magic = 5
	genesis.BlockCostEnabled = false // disable block throttling
	genesis.CustomAllocation = []*chain.CustomAllocation{
		{
			Address: sender,
			Balance: 10000000,
		},
	}
	airdropData := []byte(fmt.Sprintf(`[{"address":"%s"}]`, sender2))
	genesis.AirdropHash = ecommon.BytesToHash(crypto.Keccak256(airdropData)).Hex()
	genesis.AirdropUnits = 1000000000
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
		v := &vm.VM{AirdropData: airdropData}
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
		g, err := cli.Genesis(context.Background())
		gomega.Ω(err).Should(gomega.BeNil())

		for _, alloc := range g.CustomAllocation {
			bal, err := cli.Balance(context.Background(), alloc.Address)
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
			ok, err := cli.Ping(context.Background())
			gomega.Ω(ok).Should(gomega.BeTrue())
			gomega.Ω(err).Should(gomega.BeNil())
		}
	})
})

var _ = ginkgo.Describe("[Network]", func() {
	ginkgo.It("can get network", func() {
		for _, inst := range instances {
			cli := inst.cli
			networkID, subnetID, chainID, err := cli.Network(context.Background())
			gomega.Ω(networkID).Should(gomega.Equal(uint32(1)))
			gomega.Ω(subnetID).ShouldNot(gomega.Equal(ids.Empty))
			gomega.Ω(chainID).ShouldNot(gomega.Equal(ids.Empty))
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

var _ = ginkgo.Describe("Tx Types", func() {
	ginkgo.It("ensure activity yet", func() {
		activity, err := instances[0].cli.RecentActivity(context.Background())
		gomega.Ω(err).To(gomega.BeNil())

		gomega.Ω(len(activity)).To(gomega.Equal(0))
	})

	ginkgo.It("ensure nothing owned yet", func() {
		spaces, err := instances[0].cli.Owned(context.Background(), sender)
		gomega.Ω(err).To(gomega.BeNil())

		gomega.Ω(len(spaces)).To(gomega.Equal(0))
	})

	ginkgo.It("get currently accepted block ID", func() {
		for _, inst := range instances {
			cli := inst.cli
			_, err := cli.Accepted(context.Background())
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
			_, _, err := client.SignIssueRawTx(ctx, instances[0].cli, claimTx, priv)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("send gossip from node 0 to 1", func() {
			newTxs := instances[0].vm.Mempool().NewTxs(genesis.TargetBlockSize)
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

		ginkgo.By("ensure something owned", func() {
			spaces, err := instances[1].cli.Owned(context.Background(), sender)
			gomega.Ω(err).To(gomega.BeNil())

			gomega.Ω(spaces).To(gomega.Equal([]string{space}))
		})

		ginkgo.By("extend time with lifeline", func() {
			lifelineTx := &chain.LifelineTx{
				BaseTx: &chain.BaseTx{},
				Space:  space,
				Units:  1,
			}
			createIssueRawTx(instances[1], lifelineTx, priv)
			expectBlkAccept(instances[1])
		})

		ginkgo.By("ensure all activity accounted for", func() {
			activity, err := instances[1].cli.RecentActivity(context.Background())
			gomega.Ω(err).To(gomega.BeNil())

			gomega.Ω(len(activity)).To(gomega.Equal(2))
			a0 := activity[0]
			gomega.Ω(a0.Typ).To(gomega.Equal("lifeline"))
			gomega.Ω(a0.Space).To(gomega.Equal(space))
			gomega.Ω(a0.Units).To(gomega.Equal(uint64(1)))
			gomega.Ω(a0.Sender).To(gomega.Equal(sender.Hex()))
			a1 := activity[1]
			gomega.Ω(a1.Typ).To(gomega.Equal("claim"))
			gomega.Ω(a1.Space).To(gomega.Equal(space))
			gomega.Ω(a1.Sender).To(gomega.Equal(sender.Hex()))
		})
	})

	ginkgo.It("fail ClaimTx with no block ID", func() {
		utx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{},
			Space:  "foo",
		}

		dh, err := chain.DigestHash(utx)
		gomega.Ω(err).Should(gomega.BeNil())
		sig, err := chain.Sign(dh, priv)
		gomega.Ω(err).Should(gomega.BeNil())

		tx := chain.NewTx(utx, sig)
		err = tx.Init(genesis)
		gomega.Ω(err).Should(gomega.BeNil())

		_, err = instances[0].cli.IssueRawTx(context.Background(), tx.Bytes())
		gomega.Ω(err.Error()).Should(gomega.Equal(chain.ErrInvalidBlockID.Error()))
	})

	ginkgo.It("Claim/SetTx in a single node", func() {
		space := strings.Repeat("b", parser.MaxIdentifierSize)
		claimTx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{},
			Space:  space,
		}

		ginkgo.By("mine and accept block with the first ClaimTx", func() {
			createIssueRawTx(instances[0], claimTx, priv)
			expectBlkAccept(instances[0])
		})

		ginkgo.By("ensure everything owned", func() {
			spaces, err := instances[0].cli.Owned(context.Background(), sender)
			gomega.Ω(err).To(gomega.BeNil())

			gomega.Ω(spaces).To(gomega.ContainElements(
				strings.Repeat("a", parser.MaxIdentifierSize),
				strings.Repeat("b", parser.MaxIdentifierSize),
			))
		})

		ginkgo.By("check space after ClaimTx has been accepted", func() {
			pf, values, err := instances[0].cli.Info(context.Background(), space)
			gomega.Ω(err).To(gomega.BeNil())
			gomega.Ω(pf).NotTo(gomega.BeNil())
			gomega.Ω(pf.Units).To(gomega.Equal(uint64(100)))
			gomega.Ω(pf.Owner).To(gomega.Equal(sender))
			gomega.Ω(len(values)).To(gomega.Equal(0))
		})

		k, v := "avaxkvm", []byte("hello")
		setTx := &chain.SetTx{
			BaseTx: &chain.BaseTx{},
			Space:  space,
			Key:    k,
			Value:  v,
		}

		ginkgo.By("accept block with a new SetTx", func() {
			createIssueRawTx(instances[0], setTx, priv)
			expectBlkAccept(instances[0])
		})

		ginkgo.By("read back from VM with range query", func() {
			_, kvs, err := instances[0].cli.Info(context.Background(), space)
			gomega.Ω(err).To(gomega.BeNil())
			gomega.Ω(kvs[0].Key).To(gomega.Equal(k))
			gomega.Ω(kvs[0].ValueMeta.Size).To(gomega.Equal(uint64(5)))
		})

		ginkgo.By("read back from VM with resolve", func() {
			exists, value, valueMeta, err := instances[0].cli.Resolve(context.Background(), space+"/"+k)
			gomega.Ω(err).To(gomega.BeNil())
			gomega.Ω(exists).To(gomega.BeTrue())
			gomega.Ω(value).To(gomega.Equal(v))
			gomega.Ω(valueMeta.Size).To(gomega.Equal(uint64(5)))
		})

		ginkgo.By("transfer funds to other sender", func() {
			transferTx := &chain.TransferTx{
				BaseTx: &chain.BaseTx{},
				To:     sender2,
				Units:  100,
			}
			createIssueRawTx(instances[0], transferTx, priv)
			expectBlkAccept(instances[0])
		})

		ginkgo.By("move space to other sender", func() {
			moveTx := &chain.MoveTx{
				BaseTx: &chain.BaseTx{},
				To:     sender2,
				Space:  space,
			}
			createIssueRawTx(instances[0], moveTx, priv)
			expectBlkAccept(instances[0])
		})

		ginkgo.By("ensure all activity accounted for", func() {
			activity, err := instances[0].cli.RecentActivity(context.Background())
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

		ginkgo.By("transfer funds to other sender (simple)", func() {
			createIssueTx(instances[0], &chain.Input{
				Typ:   chain.Transfer,
				To:    sender2,
				Units: 100,
			}, priv)
			expectBlkAccept(instances[0])
		})

		ginkgo.By("move space to other sender (simple)", func() {
			createIssueTx(instances[0], &chain.Input{
				Typ:   chain.Move,
				To:    sender,
				Space: space,
			}, priv2)
			expectBlkAccept(instances[0])
		})
	})

	ginkgo.It("Distribute Lottery Reward", func() {
		ginkgo.By("ensure that sender is rewarded at least once", func() {
			bal, err := instances[0].cli.Balance(context.Background(), sender)
			gomega.Ω(err).To(gomega.BeNil())

			found := false
			for i := 0; i < 100; i++ {
				claimTx := &chain.ClaimTx{
					BaseTx: &chain.BaseTx{},
					Space:  RandStringRunes(64),
				}

				// Use a different sender so that sender is rewarded
				createIssueRawTx(instances[0], claimTx, priv2)
				expectBlkAccept(instances[0])

				bal2, err := instances[0].cli.Balance(context.Background(), sender)
				gomega.Ω(err).To(gomega.BeNil())

				if bal2 > bal {
					found = true
					break
				}
			}
			gomega.Ω(found).To(gomega.BeTrue())
		})

		ginkgo.By("ensure all activity accounted for", func() {
			activity, err := instances[0].cli.RecentActivity(context.Background())
			gomega.Ω(err).To(gomega.BeNil())

			a0 := activity[0]
			gomega.Ω(a0.Typ).To(gomega.Equal("reward"))
			gomega.Ω(a0.To).To(gomega.Equal(sender.Hex()))
			gomega.Ω(len(a0.Sender)).To(gomega.Equal(0))
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
			_, _, err := client.SignIssueRawTx(ctx, instances[0].cli, claimTx, priv)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		// since the block from previous test spec has not been replicated yet
		ginkgo.By("send gossip from node 0 to 1 should fail on server-side since 1 doesn't have the block yet", func() {
			newTxs := instances[0].vm.Mempool().NewTxs(genesis.TargetBlockSize)
			gomega.Ω(len(newTxs)).To(gomega.Equal(1))

			err := instances[0].vm.Network().GossipNewTxs(newTxs)
			gomega.Ω(err).Should(gomega.BeNil())

			// mempool in 1 should be empty, since gossip/submit failed
			gomega.Ω(instances[1].vm.Mempool().Len()).Should(gomega.Equal(0))
		})
	})

	ginkgo.It("file ops work", func() {
		space := "coolfilestorageforall"
		ginkgo.By("create space", func() {
			createIssueTx(instances[0], &chain.Input{
				Typ:   chain.Claim,
				Space: space,
			}, priv)
			expectBlkAccept(instances[0])
		})

		files := []string{}
		ginkgo.By("create 0-files", func() {
			for _, size := range []int64{units.KiB, 278 * units.KiB, 400 * units.KiB /* right on boundary */, 5 * units.MiB} {
				newFile, err := ioutil.TempFile("", "test")
				gomega.Ω(err).Should(gomega.BeNil())
				_, err = newFile.Seek(size-1, 0)
				gomega.Ω(err).Should(gomega.BeNil())
				_, err = newFile.Write([]byte{0})
				gomega.Ω(err).Should(gomega.BeNil())
				gomega.Ω(newFile.Close()).Should(gomega.BeNil())
				files = append(files, newFile.Name())
			}
		})

		ginkgo.By("create random files", func() {
			for _, size := range []int{units.KiB, 400 * units.KiB, 3 * units.MiB} {
				newFile, err := ioutil.TempFile("", "test")
				gomega.Ω(err).Should(gomega.BeNil())
				_, err = newFile.WriteString(RandStringRunes(size))
				gomega.Ω(err).Should(gomega.BeNil())
				gomega.Ω(newFile.Close()).Should(gomega.BeNil())
				files = append(files, newFile.Name())
			}
		})

		for _, file := range files {
			var path string
			var originalFile *os.File
			var err error
			ginkgo.By("upload file", func() {
				originalFile, err = os.Open(file)
				gomega.Ω(err).Should(gomega.BeNil())

				c := make(chan struct{})
				d := make(chan struct{})
				go func() {
					asyncBlockPush(instances[0], c)
					close(d)
				}()
				path, err = tree.Upload(
					context.Background(), instances[0].cli, priv,
					space, originalFile, int(genesis.MaxValueSize),
				)
				gomega.Ω(err).Should(gomega.BeNil())
				close(c)
				<-d
			})

			var newFile *os.File
			ginkgo.By("download file", func() {
				newFile, err = ioutil.TempFile("", "computer")
				gomega.Ω(err).Should(gomega.BeNil())

				err = tree.Download(context.Background(), instances[0].cli, path, newFile)
				gomega.Ω(err).Should(gomega.BeNil())
			})

			ginkgo.By("compare file contents", func() {
				_, err = originalFile.Seek(0, io.SeekStart)
				gomega.Ω(err).Should(gomega.BeNil())
				rho := sha256.New()
				_, err = io.Copy(rho, originalFile)
				gomega.Ω(err).Should(gomega.BeNil())
				ho := fmt.Sprintf("%x", rho.Sum(nil))

				_, err = newFile.Seek(0, io.SeekStart)
				gomega.Ω(err).Should(gomega.BeNil())
				rhn := sha256.New()
				_, err = io.Copy(rhn, newFile)
				gomega.Ω(err).Should(gomega.BeNil())
				hn := fmt.Sprintf("%x", rhn.Sum(nil))

				gomega.Ω(ho).Should(gomega.Equal(hn))

				originalFile.Close()
				newFile.Close()
			})

			ginkgo.By("delete file", func() {
				c := make(chan struct{})
				d := make(chan struct{})
				go func() {
					asyncBlockPush(instances[0], c)
					close(d)
				}()
				err = tree.Delete(context.Background(), instances[0].cli, path, priv)
				gomega.Ω(err).Should(gomega.BeNil())
				close(c)
				<-d

				// Should error
				dummyFile, err := ioutil.TempFile("", "computer_copy")
				gomega.Ω(err).Should(gomega.BeNil())
				err = tree.Download(context.Background(), instances[0].cli, path, dummyFile)
				gomega.Ω(err).Should(gomega.MatchError(tree.ErrMissing))
				dummyFile.Close()
			})
		}
	})

	// TODO: full replicate blocks between nodes
})

func createIssueRawTx(i instance, utx chain.UnsignedTransaction, signer *ecdsa.PrivateKey) {
	g, err := i.cli.Genesis(context.Background())
	gomega.Ω(err).Should(gomega.BeNil())
	utx.SetMagic(g.Magic)

	la, err := i.cli.Accepted(context.Background())
	gomega.Ω(err).Should(gomega.BeNil())
	utx.SetBlockID(la)

	price, blockCost, err := i.cli.SuggestedRawFee(context.Background())
	gomega.Ω(err).Should(gomega.BeNil())
	utx.SetPrice(price + blockCost/utx.FeeUnits(g))

	dh, err := chain.DigestHash(utx)
	gomega.Ω(err).Should(gomega.BeNil())
	sig, err := chain.Sign(dh, signer)
	gomega.Ω(err).Should(gomega.BeNil())

	tx := chain.NewTx(utx, sig)
	err = tx.Init(genesis)
	gomega.Ω(err).To(gomega.BeNil())

	_, err = i.cli.IssueRawTx(context.Background(), tx.Bytes())
	gomega.Ω(err).To(gomega.BeNil())
}

func createIssueTx(i instance, input *chain.Input, signer *ecdsa.PrivateKey) {
	td, _, err := i.cli.SuggestedFee(context.Background(), input)
	gomega.Ω(err).Should(gomega.BeNil())

	dh, err := tdata.DigestHash(td)
	gomega.Ω(err).Should(gomega.BeNil())

	sig, err := chain.Sign(dh, signer)
	gomega.Ω(err).Should(gomega.BeNil())

	_, err = i.cli.IssueTx(context.Background(), td, sig)
	gomega.Ω(err).To(gomega.BeNil())
}

func asyncBlockPush(i instance, c chan struct{}) {
	timer := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-c:
			return
		case <-timer.C:
			// manually signal ready
			i.builder.NotifyBuild()
			// manually ack ready sig as in engine
			<-i.toEngine

			blk, err := i.vm.BuildBlock()
			if err != nil {
				continue
			}

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
	}
}

func expectBlkAccept(i instance) {
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
