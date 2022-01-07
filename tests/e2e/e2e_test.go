// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// e2e implements the e2e tests.
package e2e_test

import (
	"context"
	"flag"
	"fmt"
	"syscall"
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/client"
	"github.com/ava-labs/quarkvm/tests"
	"github.com/fatih/color"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
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
	requestTimeout  time.Duration
	clusterInfoPath string
	minExpiry       uint64
	pruneInterval   uint64
	shutdown        bool
)

func init() {
	flag.DurationVar(
		&requestTimeout,
		"request-timeout",
		30*time.Second,
		"timeout for transaction issuance and confirmation",
	)
	flag.StringVar(
		&clusterInfoPath,
		"cluster-info-path",
		"",
		"cluster info YAML file path (as defined in 'tests/cluster_info.go')",
	)
	flag.Uint64Var(
		&minExpiry,
		"min-expiry",
		chain.DefaultMinExpiryTime,
		"minimum number of seconds to expire prefix since its block time (must be set via genesis)",
	)
	flag.Uint64Var(
		&pruneInterval,
		"prune-interval",
		chain.DefaultPruneInterval,
		"prune interval in seconds",
	)
	flag.BoolVar(
		&shutdown,
		"shutdown",
		false,
		"'true' to send SIGINT to the local cluster for shutdown",
	)
}

var (
	priv   crypto.PrivateKey
	sender [crypto.SECP256K1RPKLen]byte

	clusterInfo tests.ClusterInfo
	instances   []instance
	cur         int // index of current client
)

func next() {
	cur++
	cur %= len(instances)
}

type instance struct {
	uri string
	cli client.Client
}

var _ = ginkgo.BeforeSuite(func() {
	var err error
	priv, err = f.NewPrivateKey()
	gomega.Ω(err).Should(gomega.BeNil())
	sender, err = chain.FormatPK(priv.PublicKey())
	gomega.Ω(err).Should(gomega.BeNil())

	gomega.Ω(clusterInfoPath).ShouldNot(gomega.BeEmpty())
	clusterInfo, err = tests.LoadClusterInfo(clusterInfoPath)
	gomega.Ω(err).Should(gomega.BeNil())

	n := len(clusterInfo.URIs)
	gomega.Ω(n).Should(gomega.BeNumerically(">", 1))

	if shutdown {
		gomega.Ω(clusterInfo.PID).Should(gomega.BeNumerically(">", 1))
	}

	instances = make([]instance, n)
	for i := range instances {
		u := clusterInfo.URIs[i]
		instances[i] = instance{
			uri: u,
			cli: client.New(u, clusterInfo.Endpoint, requestTimeout),
		}
	}
	color.Blue("created clients with %+v", clusterInfo)
})

var _ = ginkgo.AfterSuite(func() {
	if !shutdown {
		color.Red("skipping shutdown for PID %d", clusterInfo.PID)
		return
	}
	color.Red("shutting down local cluster on PID %d", clusterInfo.PID)
	serr := syscall.Kill(clusterInfo.PID, syscall.SIGTERM)
	color.Red("terminated local cluster on PID %d (error %v)", clusterInfo.PID, serr)
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

var _ = ginkgo.Describe("[Claim/SetTx]", func() {
	ginkgo.It("get currently preferred block ID", func() {
		for _, inst := range instances {
			cli := inst.cli
			_, err := cli.Preferred()
			gomega.Ω(err).Should(gomega.BeNil())
		}
	})

	priv2, err := f.NewPrivateKey()
	gomega.Ω(err).Should(gomega.BeNil())
	sender2, err := chain.FormatPK(priv2.PublicKey())
	gomega.Ω(err).Should(gomega.BeNil())

	pfx := []byte(fmt.Sprintf("%10d", time.Now().UnixNano()))
	ginkgo.It("fail ClaimTx with invalid signature", func() {
		claimTx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{
				Sender: sender2,
				Prefix: pfx,
			},
			Expiry: minExpiry + 10,
		}
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		_, err = client.MineSignIssueTx(
			ctx,
			instances[cur].cli,
			claimTx,
			priv,
			client.WithPollTx(),
		)
		cancel()
		gomega.Ω(err).Should(gomega.MatchError(chain.ErrInvalidSignature.Error()))
		next()
	})

	ginkgo.It("Claim/SetTx with valid PoW", func() {
		ginkgo.By("mine and issue ClaimTx", func() {
			claimTx := &chain.ClaimTx{
				BaseTx: &chain.BaseTx{
					Sender: sender,
					Prefix: pfx,
				},
				Expiry: minExpiry,
			}
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.MineSignIssueTx(
				ctx,
				instances[cur].cli,
				claimTx,
				priv,
				client.WithPollTx(),
				client.WithPrefixInfo(pfx),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
			next()

			ctx, cancel = context.WithTimeout(context.Background(), requestTimeout)
			_, err = client.MineSignIssueTx(
				ctx,
				instances[cur].cli,
				claimTx,
				priv,
				client.WithPollTx(),
			)
			cancel()
			gomega.Ω(err).Should(gomega.MatchError(chain.ErrPrefixNotExpired.Error()))
			next()
		})

		ginkgo.By("check prefix to check if ClaimTx has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking prefix on %q", inst.uri)
				pf, err := inst.cli.PrefixInfo(pfx)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(pf.Keys).To(gomega.Equal(int64(1)))
				gomega.Ω(pf.Owner).To(gomega.Equal(sender))
			}
		})

		k, v := []byte("avax.kvm"), []byte("hello")
		ginkgo.By("mine and issue SetTx to a different node (if available)", func() {
			setTx := &chain.SetTx{
				BaseTx: &chain.BaseTx{
					Sender: sender,
					Prefix: pfx,
				},
				Key:   k,
				Value: v,
			}
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.MineSignIssueTx(
				ctx,
				instances[cur].cli,
				setTx,
				priv,
				client.WithPollTx(),
				client.WithPrefixInfo(pfx),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
			next()
		})

		ginkgo.By("check prefix to check if SetTx has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking prefix on %q", inst.uri)
				pf, err := inst.cli.PrefixInfo(pfx)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(pf.Keys).To(gomega.Equal(int64(2)))
				gomega.Ω(pf.Owner).To(gomega.Equal(sender))
			}
		})

		ginkgo.By("send Range to all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking SetTx with Range on %q", inst.uri)
				kvs, err := inst.cli.Range(pfx, k)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(kvs[0].Key).To(gomega.Equal(k))
				gomega.Ω(kvs[0].Value).To(gomega.Equal(v))
			}
		})

		ginkgo.By("can claim the same prefix after expiration", func() {
			// wait enough for vm expire next to trigger prune
			waitDur := time.Duration(minExpiry)*time.Second + time.Minute
			color.Blue("waiting %v to reach expiry with some buffer", waitDur)
			time.Sleep(waitDur)

			// trigger block creation to call "ExpireNext"
			claimTx := &chain.ClaimTx{
				BaseTx: &chain.BaseTx{
					Sender: sender,
					Prefix: []byte(fmt.Sprintf("otherpfx%d", time.Now().UnixNano())),
				},
				Expiry: minExpiry,
			}
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.MineSignIssueTx(
				ctx,
				instances[cur].cli,
				claimTx,
				priv,
				client.WithPollTx(),
				client.WithPrefixInfo(pfx),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
			next()

			// add default prune interval plus expiry for some buffer
			waitDur = time.Duration(3*pruneInterval) * time.Second
			color.Blue("waiting %v for pruning routine", waitDur)
			time.Sleep(waitDur)

			// prefix should've been deleted
			pf, err := instances[cur].cli.PrefixInfo(pfx)
			gomega.Ω(pf).Should(gomega.BeNil())
			gomega.Ω(err).Should(gomega.BeNil())
			next()
		})
	})

	ginkgo.It("fail SetTx with invalid signature", func() {
		setTx := &chain.SetTx{
			BaseTx: &chain.BaseTx{
				Sender: sender2,
				Prefix: pfx,
			},
			Key:   []byte("k"),
			Value: []byte("v"),
		}
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		_, err = client.MineSignIssueTx(
			ctx,
			instances[cur].cli,
			setTx,
			priv,
			client.WithPollTx(),
		)
		cancel()
		gomega.Ω(err).Should(gomega.MatchError(chain.ErrInvalidSignature.Error()))
		next()
	})
})

var _ = ginkgo.Describe("[Claim/LifelineTx]", func() {
	pfx := []byte(fmt.Sprintf("claimlifelinetx%10d", time.Now().UnixNano()))
	ginkgo.It("fail ClaimTx with invalid expiry", func() {
		claimTx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{
				Sender: sender,
				Prefix: pfx,
			},
			Expiry: minExpiry - 10,
		}
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		_, err := client.MineSignIssueTx(
			ctx,
			instances[cur].cli,
			claimTx,
			priv,
			client.WithPollTx(),
		)
		cancel()
		gomega.Ω(err).Should(gomega.MatchError(chain.ErrInvalidExpiry.Error()))
		next()
	})

	ginkgo.It("ClaimTx", func() {
		claimTx := &chain.ClaimTx{
			BaseTx: &chain.BaseTx{
				Sender: sender,
				Prefix: pfx,
			},
		}
		ginkgo.By("success with valid expiry", func() {
			claimTx.Expiry = minExpiry + 10
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.MineSignIssueTx(
				ctx,
				instances[cur].cli,
				claimTx,
				priv,
				client.WithPollTx(),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
			next()
		})

		ginkgo.By("once tx is confirmed, prefix info should be persisted", func() {
			pf, err := instances[cur].cli.PrefixInfo(pfx)
			gomega.Ω(err).Should(gomega.BeNil())
			gomega.Ω(pf.Keys).To(gomega.Equal(int64(1)))
			gomega.Ω(pf.Owner).To(gomega.Equal(sender))
			next()
		})

		ginkgo.By("fail when prefix is not expired yet", func() {
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.MineSignIssueTx(
				ctx,
				instances[cur].cli,
				claimTx,
				priv,
				client.WithPollTx(),
			)
			cancel()
			gomega.Ω(err).Should(gomega.MatchError(chain.ErrPrefixNotExpired.Error()))
			next()
		})

		lifelineTx := &chain.LifelineTx{
			BaseTx: &chain.BaseTx{
				Sender: sender,
				Prefix: pfx,
			},
			Expiry: minExpiry + 10,
		}
		ginkgo.By("extend prefix lease with LifelineTx before expiration", func() {
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.MineSignIssueTx(
				ctx,
				instances[cur].cli,
				lifelineTx,
				priv,
				client.WithPollTx(),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
			next()
		})

		ginkgo.By("once tx is confirmed, prefix info should be persisted", func() {
			pf, err := instances[cur].cli.PrefixInfo(pfx)
			gomega.Ω(err).Should(gomega.BeNil())
			gomega.Ω(pf.Keys).To(gomega.Equal(int64(1)))
			gomega.Ω(pf.Owner).To(gomega.Equal(sender))
			next()
		})

		ginkgo.By("once prefix is expired, prefix info should be nil", func() {
			// wait enough for vm expire next to trigger prune
			waitDur := time.Duration(2*lifelineTx.Expiry)*time.Second + time.Minute
			color.Blue("waiting %v to reach expiry with some buffer", waitDur)
			time.Sleep(waitDur)

			// trigger some block creation which calls "ExpireNext"
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.MineSignIssueTx(
				ctx,
				instances[cur].cli,
				&chain.ClaimTx{
					BaseTx: &chain.BaseTx{
						Sender: sender,
						Prefix: []byte(fmt.Sprintf("%d", time.Now().UnixNano())),
					},
					Expiry: minExpiry,
				},
				priv,
				client.WithPollTx(),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
			next()

			// add default prune interval plus expiry for some buffer
			waitDur = time.Duration(3*pruneInterval) * time.Second
			color.Blue("waiting %v for pruning routine", waitDur)
			time.Sleep(waitDur)

			pf, err := instances[cur].cli.PrefixInfo(pfx)
			gomega.Ω(pf).Should(gomega.BeNil())
			gomega.Ω(err).Should(gomega.BeNil())
			next()
		})
	})
})
