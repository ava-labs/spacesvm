// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// e2e implements the e2e tests.
package e2e_test

import (
	"bytes"
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
)

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

	pfx := []byte(fmt.Sprintf("%10d", time.Now().UnixNano()))
	ginkgo.It("Claim/SetTx with valid PoW in a single node", func() {
		ginkgo.By("mine and issue ClaimTx to the first node", func() {
			claimTx := &chain.ClaimTx{
				BaseTx: &chain.BaseTx{
					Sender: sender,
					Prefix: pfx,
				},
			}
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.MineSignIssueTx(
				ctx,
				instances[0].cli,
				claimTx,
				priv,
				client.WithPollTx(),
				client.WithPrefixInfo(pfx),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("check prefix to check if ClaimTx has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking prefix on %q", inst.uri)
				pf, err := inst.cli.PrefixInfo(pfx)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(pf.Units).To(gomega.Equal(int64(1)))
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

			cli := instances[0].cli
			if len(instances) > 1 {
				cli = instances[1].cli
			}

			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.MineSignIssueTx(
				ctx,
				cli,
				setTx,
				priv,
				client.WithPollTx(),
				client.WithPrefixInfo(pfx),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("check prefix to check if SetTx has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking prefix on %q", inst.uri)
				pf, err := inst.cli.PrefixInfo(pfx)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(pf.Units).To(gomega.Equal(int64(2)))
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

		v2 := bytes.Repeat([]byte("a"), chain.SetValueDiscount*chain.ValueUnitLength*2+1)
		ginkgo.By("mine and issue large SetTx overwrite to a different node (if available)", func() {
			setTx := &chain.SetTx{
				BaseTx: &chain.BaseTx{
					Sender: sender,
					Prefix: pfx,
				},
				Key:   k,
				Value: v2,
			}

			cli := instances[0].cli
			if len(instances) > 1 {
				cli = instances[1].cli
			}

			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.MineSignIssueTx(
				ctx,
				cli,
				setTx,
				priv,
				client.WithPollTx(),
				client.WithPrefixInfo(pfx),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("check prefix to check if SetTx has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking prefix on %q", inst.uri)
				pf, err := inst.cli.PrefixInfo(pfx)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(pf.Units).To(gomega.Equal(int64(2)))
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

		ginkgo.By("mine and issue delete SetTx to a different node (if available)", func() {
			setTx := &chain.SetTx{
				BaseTx: &chain.BaseTx{
					Sender: sender,
					Prefix: pfx,
				},
				Key: k,
			}

			cli := instances[0].cli
			if len(instances) > 1 {
				cli = instances[1].cli
			}

			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := client.MineSignIssueTx(
				ctx,
				cli,
				setTx,
				priv,
				client.WithPollTx(),
				client.WithPrefixInfo(pfx),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("check prefix to check if SetTx has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking prefix on %q", inst.uri)
				pf, err := inst.cli.PrefixInfo(pfx)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(pf.Units).To(gomega.Equal(int64(2)))
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
	})
})
