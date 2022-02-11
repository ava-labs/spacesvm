// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// e2e implements the e2e tests.
package e2e_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"flag"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	ecommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/parser"
	"github.com/ava-labs/spacesvm/tests"
)

func TestIntegration(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "spacesvm integration test suites")
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
		120*time.Second,
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
	priv   *ecdsa.PrivateKey
	sender ecommon.Address

	clusterInfo tests.ClusterInfo
	instances   []instance

	genesis *chain.Genesis
)

type instance struct {
	uri string
	cli client.Client
}

var _ = ginkgo.BeforeSuite(func() {
	var err error
	priv, err = crypto.HexToECDSA("a1c0bd71ff64aebd666b04db0531d61479c2c031e4de38410de0609cbd6e66f0")
	gomega.Ω(err).Should(gomega.BeNil())
	sender = crypto.PubkeyToAddress(priv.PublicKey)

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
		u := clusterInfo.URIs[i] + clusterInfo.Endpoint
		instances[i] = instance{
			uri: u,
			cli: client.New(u, requestTimeout),
		}
	}
	genesis, err = instances[0].cli.Genesis(context.Background())
	gomega.Ω(err).Should(gomega.BeNil())
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
			ok, err := cli.Ping(context.Background())
			gomega.Ω(ok).Should(gomega.BeTrue())
			gomega.Ω(err).Should(gomega.BeNil())
		}
	})
})

var _ = ginkgo.Describe("[Network]", func() {
	ginkgo.It("can get network", func() {
		sID, err := ids.FromString("24tZhrm8j8GCJRE9PomW8FaeqbgGS4UAQjJnqqn8pq5NwYSYV1")
		gomega.Ω(err).Should(gomega.BeNil())
		for _, inst := range instances {
			cli := inst.cli
			networkID, subnetID, chainID, err := cli.Network(context.Background())
			gomega.Ω(networkID).Should(gomega.Equal(uint32(1337)))
			gomega.Ω(subnetID).Should(gomega.Equal(sID))
			gomega.Ω(chainID).ShouldNot(gomega.Equal(ids.Empty))
			gomega.Ω(err).Should(gomega.BeNil())
		}
	})
})

var _ = ginkgo.Describe("[Claim/SetTx]", func() {
	ginkgo.It("get currently accepted block ID", func() {
		for _, inst := range instances {
			cli := inst.cli
			_, err := cli.Accepted(context.Background())
			gomega.Ω(err).Should(gomega.BeNil())
		}
	})

	ginkgo.It("Claim/SetTx in a single node (raw)", func() {
		space := strings.Repeat("a", parser.MaxIdentifierSize)
		ginkgo.By("mine and issue ClaimTx to the first node", func() {
			claimTx := &chain.ClaimTx{
				BaseTx: &chain.BaseTx{},
				Space:  space,
			}

			claimed, err := instances[0].cli.Claimed(context.Background(), space)
			gomega.Ω(err).Should(gomega.BeNil())
			gomega.Ω(claimed).Should(gomega.BeFalse())

			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, _, err = client.SignIssueRawTx(
				ctx,
				instances[0].cli,
				claimTx,
				priv,
				client.WithPollTx(),
				client.WithInfo(space),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())

			claimed, err = instances[0].cli.Claimed(context.Background(), space)
			gomega.Ω(err).Should(gomega.BeNil())
			gomega.Ω(claimed).Should(gomega.BeTrue())
		})

		ginkgo.By("check space to check if ClaimTx has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking space on %q", inst.uri)
				pf, _, err := inst.cli.Info(context.Background(), space)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(pf.Units).To(gomega.Equal(uint64(100)))
				gomega.Ω(pf.Owner).To(gomega.Equal(sender))
			}
		})

		k, v := "avaxkvm", []byte("hello")
		ginkgo.By("mine and issue SetTx to a different node (if available)", func() {
			setTx := &chain.SetTx{
				BaseTx: &chain.BaseTx{},
				Space:  space,
				Key:    k,
				Value:  v,
			}

			cli := instances[0].cli
			if len(instances) > 1 {
				cli = instances[1].cli
			}

			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, _, err := client.SignIssueRawTx(
				ctx,
				cli,
				setTx,
				priv,
				client.WithPollTx(),
				client.WithInfo(space),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("check space to check if SetTx has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking space on %q", inst.uri)
				pf, _, err := inst.cli.Info(context.Background(), space)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(pf.Units).To(gomega.Equal(uint64(100)))
				gomega.Ω(pf.Owner).To(gomega.Equal(sender))
			}
		})

		ginkgo.By("send Range to all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking SetTx with Range on %q", inst.uri)
				_, kvs, err := inst.cli.Info(context.Background(), space)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(kvs[0].Key).To(gomega.Equal(k))
				gomega.Ω(kvs[0].ValueMeta.Size).To(gomega.Equal(uint64(len(v))))
				_, rv, _, err := inst.cli.Resolve(context.Background(), space+"/"+kvs[0].Key)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(rv).To(gomega.Equal(v))
			}
		})

		v2 := bytes.Repeat([]byte("a"), int(genesis.ValueUnitSize)*20+1)
		ginkgo.By("mine and issue large SetTx overwrite to a different node (if available)", func() {
			setTx := &chain.SetTx{
				BaseTx: &chain.BaseTx{},
				Space:  space,
				Key:    k,
				Value:  v2,
			}

			cli := instances[0].cli
			if len(instances) > 1 {
				cli = instances[1].cli
			}

			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, _, err := client.SignIssueRawTx(
				ctx,
				cli,
				setTx,
				priv,
				client.WithPollTx(),
				client.WithInfo(space),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("check space to check if SetTx has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking space on %q", inst.uri)
				pf, _, err := inst.cli.Info(context.Background(), space)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(pf.Units).To(gomega.Equal(uint64(102)))
				gomega.Ω(pf.Owner).To(gomega.Equal(sender))
			}
		})

		ginkgo.By("send Range to all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking SetTx with Range on %q", inst.uri)
				_, kvs, err := inst.cli.Info(context.Background(), space)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(kvs[0].Key).To(gomega.Equal(k))
				gomega.Ω(kvs[0].ValueMeta.Size).To(gomega.Equal(uint64(len(v2))))
				_, rv, _, err := inst.cli.Resolve(context.Background(), space+"/"+kvs[0].Key)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(rv).To(gomega.Equal(v2))
			}
		})

		ginkgo.By("mine and issue delete SetTx to a different node (if available)", func() {
			deleteTx := &chain.DeleteTx{
				BaseTx: &chain.BaseTx{},
				Space:  space,
				Key:    k,
			}

			cli := instances[0].cli
			if len(instances) > 1 {
				cli = instances[1].cli
			}

			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, _, err := client.SignIssueRawTx(
				ctx,
				cli,
				deleteTx,
				priv,
				client.WithPollTx(),
				client.WithInfo(space),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("check space to check if SetTx has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking space on %q", inst.uri)
				pf, _, err := inst.cli.Info(context.Background(), space)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(pf.Units).To(gomega.Equal(uint64(100)))
				gomega.Ω(pf.Owner).To(gomega.Equal(sender))
			}
		})

		ginkgo.By("send Range to all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking SetTx with Range on %q", inst.uri)
				_, kvs, err := inst.cli.Info(context.Background(), space)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(len(kvs)).To(gomega.Equal(0))
			}
		})
	})

	ginkgo.It("Claim/SetTx in a single node", func() {
		// TODO: repeat above without copying all code
		space := strings.Repeat("b", parser.MaxIdentifierSize)
		ginkgo.By("mine and issue ClaimTx to the first node", func() {
			claimed, err := instances[0].cli.Claimed(context.Background(), space)
			gomega.Ω(err).Should(gomega.BeNil())
			gomega.Ω(claimed).Should(gomega.BeFalse())

			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, _, err = client.SignIssueTx(
				ctx,
				instances[0].cli,
				&chain.Input{
					Typ:   chain.Claim,
					Space: space,
				},
				priv,
				client.WithPollTx(),
				client.WithInfo(space),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())

			claimed, err = instances[0].cli.Claimed(context.Background(), space)
			gomega.Ω(err).Should(gomega.BeNil())
			gomega.Ω(claimed).Should(gomega.BeTrue())
		})

		ginkgo.By("check space to check if ClaimTx has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking space on %q", inst.uri)
				pf, _, err := inst.cli.Info(context.Background(), space)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(pf.Units).To(gomega.Equal(uint64(100)))
				gomega.Ω(pf.Owner).To(gomega.Equal(sender))
			}
		})

		k, v := "avaxkvm", []byte("hello")
		ginkgo.By("mine and issue SetTx to a different node (if available)", func() {
			cli := instances[0].cli
			if len(instances) > 1 {
				cli = instances[1].cli
			}

			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, _, err := client.SignIssueTx(
				ctx,
				cli,
				&chain.Input{
					Typ:   chain.Set,
					Space: space,
					Key:   k,
					Value: v,
				},
				priv,
				client.WithPollTx(),
				client.WithInfo(space),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("check space to check if SetTx has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking space on %q", inst.uri)
				pf, _, err := inst.cli.Info(context.Background(), space)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(pf.Units).To(gomega.Equal(uint64(100)))
				gomega.Ω(pf.Owner).To(gomega.Equal(sender))
			}
		})

		ginkgo.By("send Range to all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking SetTx with Range on %q", inst.uri)
				_, kvs, err := inst.cli.Info(context.Background(), space)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(kvs[0].Key).To(gomega.Equal(k))
				gomega.Ω(kvs[0].ValueMeta.Size).To(gomega.Equal(uint64(len(v))))
				_, rv, _, err := inst.cli.Resolve(context.Background(), space+"/"+kvs[0].Key)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(rv).To(gomega.Equal(v))
			}
		})

		v2 := bytes.Repeat([]byte("a"), int(genesis.ValueUnitSize)*20+1)
		ginkgo.By("mine and issue large SetTx overwrite to a different node (if available)", func() {
			cli := instances[0].cli
			if len(instances) > 1 {
				cli = instances[1].cli
			}

			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, _, err := client.SignIssueTx(
				ctx,
				cli,
				&chain.Input{
					Typ:   chain.Set,
					Space: space,
					Key:   k,
					Value: v2,
				},
				priv,
				client.WithPollTx(),
				client.WithInfo(space),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("check space to check if SetTx has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking space on %q", inst.uri)
				pf, _, err := inst.cli.Info(context.Background(), space)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(pf.Units).To(gomega.Equal(uint64(102)))
				gomega.Ω(pf.Owner).To(gomega.Equal(sender))
			}
		})

		ginkgo.By("send Range to all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking SetTx with Range on %q", inst.uri)
				_, kvs, err := inst.cli.Info(context.Background(), space)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(kvs[0].Key).To(gomega.Equal(k))
				gomega.Ω(kvs[0].ValueMeta.Size).To(gomega.Equal(uint64(len(v2))))
				_, rv, _, err := inst.cli.Resolve(context.Background(), space+"/"+kvs[0].Key)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(rv).To(gomega.Equal(v2))
			}
		})

		ginkgo.By("mine and issue delete SetTx to a different node (if available)", func() {
			cli := instances[0].cli
			if len(instances) > 1 {
				cli = instances[1].cli
			}

			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, _, err := client.SignIssueTx(
				ctx,
				cli,
				&chain.Input{
					Typ:   chain.Delete,
					Space: space,
					Key:   k,
				},
				priv,
				client.WithPollTx(),
				client.WithInfo(space),
			)
			cancel()
			gomega.Ω(err).Should(gomega.BeNil())
		})

		ginkgo.By("check space to check if SetTx has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking space on %q", inst.uri)
				pf, _, err := inst.cli.Info(context.Background(), space)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(pf.Units).To(gomega.Equal(uint64(100)))
				gomega.Ω(pf.Owner).To(gomega.Equal(sender))
			}
		})

		ginkgo.By("send Range to all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking SetTx with Range on %q", inst.uri)
				_, kvs, err := inst.cli.Info(context.Background(), space)
				gomega.Ω(err).To(gomega.BeNil())
				gomega.Ω(len(kvs)).To(gomega.Equal(0))
			}
		})
	})
})
