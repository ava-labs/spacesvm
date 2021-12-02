// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package put implements "put" commands.
package put

import (
	// "context"
	"fmt"
	"os"
	// "strings"
	"time"

	// "github.com/ava-labs/avalanchego/utils/rpc"
	// "github.com/fatih/color"
	"github.com/spf13/cobra"
	// "github.com/ava-labs/quarkvm/chain"
	// "github.com/ava-labs/quarkvm/cmd/quarkcli/create"
	// "github.com/ava-labs/quarkvm/vm"
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
		Use:   "put [options] <key> <value>",
		Short: "Puts the given key-value pair into the store",
		Long: `
Puts the given key into the store.

# prefix will be automatically parsed with delimiter "/"
# "jim" is the prefix (namespace)
# "foo" is the key
# "hello world" is the value
$ quark-cli put jim/foo "hello world"

`,
		RunE: putFunc,
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

// TODO: move all this to a separate client code
func putFunc(cmd *cobra.Command, args []string) error {
	// priv, err := create.LoadPK(privateKeyFile)
	// if err != nil {
	// 	return err
	// }

	// k, v := getPutOp(args)

	// if !strings.HasPrefix(endpoint, "/") {
	// 	endpoint = "/" + endpoint
	// }
	// color.Blue("creating requester with URL %s and endpoint %q", url, endpoint)
	// requester := rpc.NewEndpointRequester(
	// 	url,
	// 	endpoint,
	// 	"quarkvm",
	// 	requestTimeout,
	// )

	// // create unsigned transaction
	// // don't string case pubkey
	// // after grpc hop, 32 bytes becomes 64, causing
	// // panic: ed25519: bad public key length: 64
	// utx := transaction.Unsigned{
	// 	PublicKey: priv.PublicKey().Bytes(),
	// 	Op:        "Put",
	// 	Key:       k,
	// 	Value:     v,
	// }

	// // sign the unsigned transaction
	// sig, err := priv.Sign(utx.Bytes())
	// if err != nil {
	// 	return err
	// }

	// // create transaction
	// tx := &transaction.Transaction{
	// 	Unsigned:  utx,
	// 	Signature: sig,
	// }

	// // issue the transaction over tx
	// resp := new(vm.IssueTxReply)
	// if err := requester.SendRequest(
	// 	"issueTx",
	// 	&vm.IssueTxArgs{Transaction: tx},
	// 	resp,
	// ); err != nil {
	// 	color.Red("failed to issue transaction %v", err)
	// 	return err
	// }

	// txID := resp.TxID
	// color.Green("issued transaction %s (success %v)", txID, resp.Success)
	// if !resp.Success {
	// 	return fmt.Errorf("tx %v failed", txID)
	// }

	// color.Yellow("polling transaction %q", txID)
	// ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	// defer cancel()
	//ndo // ne:
	// for ctx.Err() == nil {
	// 	select {
	// 	case <-time.After(5 * time.Second):
	// 	case <-ctx.Done():
	// 		break done
	// 	}

	// 	resp := new(vm.CheckTxReply)
	// 	if err := requester.SendRequest(
	// 		"checkTx",
	// 		&vm.CheckTxArgs{TxID: txID},
	// 		resp,
	// 	); err != nil {
	// 		color.Red("polling transaction failed %v", err)
	// 	}
	// 	if resp.Error != nil {
	// 		color.Red("polling transaction error %v", resp.Error)
	// 	}
	// 	if resp.Confirmed {
	// 		color.Yellow("confirmed transaction %q", txID)
	// 		break
	// 	}
	// }
	return nil
}

func getPutOp(args []string) (key string, value string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "expected 2 arguments, got %d\n", len(args))
		os.Exit(128)
	}
	return args[0], args[1]
}
