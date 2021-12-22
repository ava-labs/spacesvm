// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// e2e implements the e2e tests.
package e2e_test

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/client"
	"github.com/ava-labs/quarkvm/crypto"
	"github.com/ava-labs/quarkvm/parser"
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
	uris           []string
	u              string
	endpoint       string
)

func init() {
	flag.DurationVar(
		&requestTimeout,
		"request-timeout",
		30*time.Second,
		"timeout for transaction issuance and confirmation",
	)
	flag.StringVar(
		&u,
		"uris",
		"",
		"comma-separated quarkvm URIs (e.g., http://127.0.0.1:9650,http://127.0.0.1:9652)",
	)
	flag.StringVar(
		&endpoint,
		"endpoint",
		"",
		"quarkvm API endpoint (e.g., /ext/bc/Bbx6eyUCSzoQLzBbM9gnLDdA9HeuiobqQS53iEthvQzeVqbwa)",
	)
}

var (
	priv *crypto.PrivateKey

	instances []instance
)

type instance struct {
	uri string
	cli client.Client
}

var _ = ginkgo.BeforeSuite(func() {
	gomega.Ω(endpoint).ShouldNot(gomega.BeEmpty())

	uris = strings.Split(u, ",")
	n := len(uris)
	gomega.Ω(n).Should(gomega.BeNumerically(">", 1))

	var err error
	priv, err = crypto.NewPrivateKey()
	gomega.Ω(err).Should(gomega.BeNil())

	instances = make([]instance, n)
	for i := range instances {
		instances[i] = instance{
			uri: uris[i],
			cli: client.New(uris[i], endpoint, requestTimeout),
		}
	}
	color.Blue("created clients with %v", uris)
})

var _ = ginkgo.AfterSuite(func() {
	// no-op
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
					Sender: priv.PublicKey().Bytes(),
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
				gomega.Ω(pf.Keys).To(gomega.Equal(int64(1)))
				gomega.Ω(pf.Owner).To(gomega.Equal(priv.PublicKey().Bytes()))
			}
		})

		k, v := []byte("avax.kvm"), []byte("hello")
		ginkgo.By("mine and issue SetTx to a different node (if available)", func() {
			setTx := &chain.SetTx{
				BaseTx: &chain.BaseTx{
					Sender: priv.PublicKey().Bytes(),
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
				gomega.Ω(pf.Keys).To(gomega.Equal(int64(2)))
				gomega.Ω(pf.Owner).To(gomega.Equal(priv.PublicKey().Bytes()))
			}
		})

		ginkgo.By("send Range to all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking SetTx with Range on %q", inst.uri)
				kvs, err := inst.cli.Range(pfx, k)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(kvs[0].Key).To(gomega.Equal(append(append(pfx, parser.Delimiter), k...)))
				gomega.Ω(kvs[0].Value).To(gomega.Equal(v))
			}
		})
	})
})
