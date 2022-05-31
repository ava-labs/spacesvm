// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// e2e implements the e2e tests.
package e2e_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	runner_sdk "github.com/ava-labs/avalanche-network-runner-sdk"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/parser"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/formatter"
	"github.com/onsi/gomega"
	"sigs.k8s.io/yaml"
)

func TestE2e(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "spacesvm e2e test suites")
}

var (
	requestTimeout time.Duration

	networkRunnerLogLevel string
	gRPCEp                string
	gRPCGatewayEp         string

	execPath  string
	pluginDir string
	logLevel  string

	vmGenesisPath string
	outputPath    string

	mode string
)

func init() {
	flag.DurationVar(
		&requestTimeout,
		"request-timeout",
		120*time.Second,
		"timeout for transaction issuance and confirmation",
	)

	flag.StringVar(
		&networkRunnerLogLevel,
		"network-runner-log-level",
		"info",
		"gRPC server endpoint",
	)
	flag.StringVar(
		&gRPCEp,
		"network-runner-grpc-endpoint",
		"0.0.0.0:8080",
		"gRPC server endpoint",
	)
	flag.StringVar(
		&gRPCGatewayEp,
		"network-runner-grpc-gateway-endpoint",
		"0.0.0.0:8081",
		"gRPC gateway endpoint",
	)

	flag.StringVar(
		&execPath,
		"avalanchego-path",
		"",
		"avalanchego executable path",
	)
	flag.StringVar(
		&logLevel,
		"avalanchego-log-level",
		"INFO",
		"avalanchego log level",
	)
	flag.StringVar(
		&pluginDir,
		"avalanchego-plugin-dir",
		"",
		"avalanchego plugin directory",
	)
	flag.StringVar(
		&vmGenesisPath,
		"vm-genesis-path",
		"",
		"VM genesis file path",
	)
	flag.StringVar(
		&outputPath,
		"output-path",
		"",
		"output YAML path to write local cluster information",
	)

	flag.StringVar(
		&mode,
		"mode",
		"test",
		"'test' to shut down cluster after tests, 'run' to skip tests and only run without shutdown",
	)
}

const vmName = "spacesvm"

var vmID ids.ID

func init() {
	// TODO: add "getVMID" util function in avalanchego and import from "avalanchego"
	b := make([]byte, 32)
	copy(b, []byte(vmName))
	var err error
	vmID, err = ids.ToID(b)
	if err != nil {
		panic(err)
	}
}

const (
	modeTest = "test"
	modeRun  = "run"
)

var (
	cli            runner_sdk.Client
	spacesvmRPCEps []string
)

var _ = ginkgo.BeforeSuite(func() {
	gomega.Expect(mode).Should(gomega.Or(gomega.Equal("test"), gomega.Equal("run")))

	var err error
	cli, err = runner_sdk.New(runner_sdk.Config{
		LogLevel:    networkRunnerLogLevel,
		Endpoint:    gRPCEp,
		DialTimeout: 10 * time.Second,
	})
	gomega.Expect(err).Should(gomega.BeNil())

	ginkgo.By("calling start API via network runner", func() {
		outf("{{green}}sending 'start' with binary path:{{/}} %q (%q)\n", execPath, vmID)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		resp, err := cli.Start(
			ctx,
			execPath,
			runner_sdk.WithLogLevel(logLevel),
			runner_sdk.WithPluginDir(pluginDir),
			runner_sdk.WithCustomVMs(map[string]string{
				vmName: vmGenesisPath,
			}))
		cancel()
		gomega.Expect(err).Should(gomega.BeNil())
		outf("{{green}}successfully started:{{/}} %+v\n", resp.ClusterInfo.NodeNames)
	})

	// TODO: network runner health should imply custom VM healthiness
	// or provide a separate API for custom VM healthiness
	// "start" is async, so wait some time for cluster health
	outf("\n{{magenta}}sleeping before checking custom VM status...{{/}}: %s\n", vmID)
	time.Sleep(3 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	_, err = cli.Health(ctx)
	cancel()
	gomega.Expect(err).Should(gomega.BeNil())

	spacesvmRPCEps = make([]string, 0)
	blockchainID, logsDir := "", ""

	// wait up to 5-minute for custom VM installation
	outf("\n{{magenta}}waiting for all custom VMs to report healthy...{{/}}\n")
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
done:
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			break done
		case <-time.After(5 * time.Second):
		}

		outf("{{magenta}}checking custom VM status{{/}}\n")
		cctx, ccancel := context.WithTimeout(context.Background(), 2*time.Minute)
		resp, err := cli.Status(cctx)
		ccancel()
		gomega.Expect(err).Should(gomega.BeNil())

		// all logs are stored under root data dir
		logsDir = resp.GetClusterInfo().GetRootDataDir()

		if v, ok := resp.ClusterInfo.CustomVms[vmID.String()]; ok {
			blockchainID = v.BlockchainId
			outf("{{blue}}spacesvm is ready:{{/}} %+v\n", v)
			break done
		}
	}
	gomega.Expect(ctx.Err()).Should(gomega.BeNil())
	cancel()

	gomega.Expect(blockchainID).Should(gomega.Not(gomega.BeEmpty()))
	gomega.Expect(logsDir).Should(gomega.Not(gomega.BeEmpty()))

	cctx, ccancel := context.WithTimeout(context.Background(), 2*time.Minute)
	uris, err := cli.URIs(cctx)
	ccancel()
	gomega.Expect(err).Should(gomega.BeNil())
	outf("{{blue}}avalanche HTTP RPCs URIs:{{/}} %q\n", uris)

	for _, u := range uris {
		rpcEP := fmt.Sprintf("%s/ext/bc/%s/rpc", u, blockchainID)
		spacesvmRPCEps = append(spacesvmRPCEps, rpcEP)
		outf("{{blue}}avalanche spacesvm RPC:{{/}} %q\n", rpcEP)
	}

	pid := os.Getpid()
	outf("{{blue}}{{bold}}writing output %q with PID %d{{/}}\n", outputPath, pid)
	ci := clusterInfo{
		URIs:     uris,
		Endpoint: fmt.Sprintf("/ext/bc/%s", blockchainID),
		PID:      pid,
		LogsDir:  logsDir,
	}
	gomega.Expect(ci.Save(outputPath)).Should(gomega.BeNil())

	b, err := os.ReadFile(outputPath)
	gomega.Expect(err).Should(gomega.BeNil())
	outf("\n{{blue}}$ cat %s:{{/}}\n%s\n", outputPath, string(b))

	priv, err = crypto.HexToECDSA("a1c0bd71ff64aebd666b04db0531d61479c2c031e4de38410de0609cbd6e66f0")
	gomega.Ω(err).Should(gomega.BeNil())
	sender = crypto.PubkeyToAddress(priv.PublicKey)

	instances = make([]instance, len(uris))
	for i := range uris {
		u := uris[i] + fmt.Sprintf("/ext/bc/%s", blockchainID)
		instances[i] = instance{
			uri: u,
			cli: client.New(u, requestTimeout),
		}
	}
	genesis, err = instances[0].cli.Genesis(context.Background())
	gomega.Ω(err).Should(gomega.BeNil())
})

var (
	priv   *ecdsa.PrivateKey
	sender eth_common.Address

	instances []instance

	genesis *chain.Genesis
)

type instance struct {
	uri string
	cli client.Client
}

var _ = ginkgo.AfterSuite(func() {
	switch mode {
	case modeTest:
		outf("{{red}}shutting down cluster{{/}}\n")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		_, err := cli.Stop(ctx)
		cancel()
		gomega.Expect(err).Should(gomega.BeNil())

	case modeRun:
		outf("{{yellow}}skipping shutting down cluster{{/}}\n")
	}

	outf("{{red}}shutting down client{{/}}\n")
	gomega.Expect(cli.Close()).Should(gomega.BeNil())
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
			networkID, _, chainID, err := cli.Network(context.Background())
			gomega.Ω(networkID).Should(gomega.Equal(uint32(1337)))
			gomega.Ω(chainID).ShouldNot(gomega.Equal(ids.Empty))
			gomega.Ω(err).Should(gomega.BeNil())
		}
	})
})

var _ = ginkgo.Describe("[Claim/SetTx]", func() {
	switch mode {
	case modeRun:
		outf("{{yellow}}skipping ClaimTx and SetTx tests{{/}}\n")
		return
	}

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

// Outputs to stdout.
//
// e.g.,
//   Out("{{green}}{{bold}}hi there %q{{/}}", "aa")
//   Out("{{magenta}}{{bold}}hi therea{{/}} {{cyan}}{{underline}}b{{/}}")
//
// ref.
// https://github.com/onsi/ginkgo/blob/v2.0.0/formatter/formatter.go#L52-L73
//
func outf(format string, args ...interface{}) {
	s := formatter.F(format, args...)
	fmt.Fprint(formatter.ColorableStdOut, s)
}

// clusterInfo represents the local cluster information.
type clusterInfo struct {
	URIs     []string `json:"uris"`
	Endpoint string   `json:"endpoint"`
	PID      int      `json:"pid"`
	LogsDir  string   `json:"logsDir"`
}

const fsModeWrite = 0o600

func (ci clusterInfo) Save(p string) error {
	ob, err := yaml.Marshal(ci)
	if err != nil {
		return err
	}
	return os.WriteFile(p, ob, fsModeWrite)
}
