package claim

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/cmd/quarkcli/create"
	"github.com/ava-labs/quarkvm/pow"
	"github.com/ava-labs/quarkvm/vm"
)

func init() {
	cobra.EnablePrefixMatching = true
}

var (
	privateKeyFile string
	url            string
	endpoint       string
	requestTimeout time.Duration
)

// NewCommand implements "quark-cli" command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim [options] <prefix>",
		Short: "Claims the given prefix",
		RunE:  claimFunc,
	}
	cmd.PersistentFlags().StringVar(
		&privateKeyFile,
		"private-key-file",
		".quark-cli-pk",
		"private key file path",
	)
	cmd.PersistentFlags().StringVar(
		&url,
		"url",
		"http://127.0.0.1:9650",
		"RPC URL for VM",
	)
	cmd.PersistentFlags().StringVar(
		&endpoint,
		"endpoint",
		"",
		"RPC endpoint for VM",
	)
	cmd.PersistentFlags().DurationVar(
		&requestTimeout,
		"request-timeout",
		30*time.Second,
		"set it to 0 to not wait for transaction confirmation",
	)
	return cmd
}

func currBlock(requester rpc.EndpointRequester) (ids.ID, error) {
	resp := new(vm.CurrBlockReply)
	if err := requester.SendRequest(
		"currBlock",
		&vm.CurrBlockArgs{},
		resp,
	); err != nil {
		color.Red("failed to get curr block %v", err)
		return ids.ID{}, err
	}
	return resp.BlockID, nil
}

func validBlockID(requester rpc.EndpointRequester, blkID ids.ID) (bool, error) {
	resp := new(vm.ValidBlockIDReply)
	if err := requester.SendRequest(
		"validBlockID",
		&vm.ValidBlockIDArgs{BlockID: blkID},
		resp,
	); err != nil {
		color.Red("failed to check valid block ID %v", err)
		return false, err
	}
	return resp.Valid, nil
}

func difficultyEstimate(requester rpc.EndpointRequester) (uint64, error) {
	resp := new(vm.DifficultyEstimateReply)
	if err := requester.SendRequest(
		"difficultyEstimate",
		&vm.DifficultyEstimateArgs{},
		resp,
	); err != nil {
		color.Red("failed to get difficulty %v", err)
		return 0, err
	}
	return resp.Difficulty, nil
}

func prefixInfo(requester rpc.EndpointRequester, prefix []byte) (*chain.PrefixInfo, error) {
	resp := new(vm.PrefixInfoReply)
	if err := requester.SendRequest(
		"prefixInfo",
		&vm.PrefixInfoArgs{Prefix: prefix},
		resp,
	); err != nil {
		color.Red("failed to get prefix %v", err)
		return nil, err
	}
	return resp.Info, nil
}

// TODO: handle timeout
func mine(requester rpc.EndpointRequester, utx chain.UnsignedTransaction) (chain.UnsignedTransaction, error) {
	for {
		cbID, err := currBlock(requester)
		if err != nil {
			return nil, err
		}
		utx.SetBlockID(cbID)
		graffiti := uint64(0)
		for {
			v, err := validBlockID(requester, cbID)
			if err != nil {
				return nil, err
			}
			if !v {
				color.Yellow("%v is no longer a valid block id", cbID)
				break
			}
			utx.SetGraffiti(graffiti)
			b, err := chain.UnsignedBytes(utx)
			if err != nil {
				return nil, err
			}
			d := pow.Difficulty(b)
			est, err := difficultyEstimate(requester)
			if err != nil {
				return nil, err
			}
			if d >= est {
				return utx, nil
			}
			graffiti++
		}
		// Get new block hash if no longer valid
	}
}

// TODO: move all this to a separate client code
func claimFunc(cmd *cobra.Command, args []string) error {
	priv, err := create.LoadPK(privateKeyFile)
	if err != nil {
		return err
	}
	prefix := getClaimOp(args)
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	color.Blue("creating requester with URL %s and endpoint %q", url, endpoint)
	requester := rpc.NewEndpointRequester(
		url,
		endpoint,
		"quarkvm",
		requestTimeout,
	)

	utx := &chain.ClaimTx{
		BaseTx: &chain.BaseTx{
			Sender: priv.PublicKey().Bytes(),
			Prefix: []byte(prefix),
		},
	}
	// TODO: make this a shared lib
	mtx, err := mine(requester, utx)
	if err != nil {
		return err
	}
	b, err := chain.UnsignedBytes(mtx)
	if err != nil {
		return err
	}
	sig, err := priv.Sign(b)
	if err != nil {
		return err
	}
	tx := chain.NewTx(mtx, sig)
	color.Yellow("Submitting tx %s with BlockID (%s): %v", tx.ID(), mtx.GetBlockID(), tx)

	resp := new(vm.IssueTxReply)
	if err := requester.SendRequest(
		"issueTx",
		&vm.IssueTxArgs{Tx: tx.Bytes()},
		resp,
	); err != nil {
		color.Red("failed to issue transaction %v", err)
		return err
	}

	txID := resp.TxID
	color.Green("issued transaction %s (success %v)", txID, resp.Success)
	if !resp.Success {
		return fmt.Errorf("tx %v failed", txID)
	}

	color.Yellow("polling transaction %q", txID)
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
done:
	for ctx.Err() == nil {
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			break done
		}

		resp := new(vm.CheckTxReply)
		if err := requester.SendRequest(
			"checkTx",
			&vm.CheckTxArgs{TxID: txID},
			resp,
		); err != nil {
			color.Red("polling transaction failed %v", err)
		}
		if resp.Confirmed {
			color.Yellow("confirmed transaction %q", txID)
			break
		}
	}

	info, err := prefixInfo(requester, []byte(prefix))
	if err != nil {
		color.Red("cannot get prefix info %v", err)
	}
	color.Blue("prefix %s info %+v", prefix, info)

	return nil
}

func getClaimOp(args []string) string {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "expected 1 arguments, got %d\n", len(args))
		os.Exit(128)
	}
	return args[0]
}
