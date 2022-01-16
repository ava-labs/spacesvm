// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package tree

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
	"github.com/ava-labs/spacesvm/parser"
)

type Root struct {
	Children []string `json:"children"`
}

func Upload(
	ctx context.Context, cli client.Client, priv *ecdsa.PrivateKey,
	space string, f io.Reader, chunkSize int,
) (string, error) {
	hashes := []string{}
	chunk := make([]byte, chunkSize)
	shouldExit := false
	opts := []client.OpOption{client.WithPollTx()}
	totalCost := uint64(0)
	for !shouldExit {
		read, err := f.Read(chunk)
		if err != nil {
			return "", err
		}
		if read == 0 {
			break
		}
		if read < chunkSize {
			shouldExit = true
			chunk = chunk[:read]
		}
		k := strings.ToLower(common.Bytes2Hex(crypto.Keccak256(chunk)))
		tx := &chain.SetTx{
			BaseTx: &chain.BaseTx{},
			Space:  space,
			Key:    k,
			Value:  chunk,
		}
		txID, cost, err := client.SignIssueRawTx(ctx, cli, tx, priv, opts...)
		if err != nil {
			return "", err
		}
		totalCost += cost
		color.Yellow("uploaded k=%s txID=%s cost=%d totalCost=%d", k, txID, cost, totalCost)
		hashes = append(hashes, k)
	}
	if len(hashes) == 0 {
		return "", ErrEmpty
	}

	rb, err := json.Marshal(&Root{
		Children: hashes,
	})
	if err != nil {
		return "", err
	}
	rk := strings.ToLower(common.Bytes2Hex(crypto.Keccak256(rb)))
	tx := &chain.SetTx{
		BaseTx: &chain.BaseTx{},
		Space:  space,
		Key:    rk,
		Value:  rb,
	}
	txID, cost, err := client.SignIssueRawTx(ctx, cli, tx, priv, opts...)
	if err != nil {
		return "", err
	}
	totalCost += cost
	color.Green("uploaded root=%s txID=%s cost=%d totalCost=%d", rk, txID, cost, totalCost)
	return space + parser.Delimiter + rk, nil
}

// TODO: make multi-threaded
func Download(cli client.Client, path string, f io.Writer) error {
	exists, rb, err := cli.Resolve(path)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w:%s", ErrMissing, path)
	}
	var r Root
	if err := json.Unmarshal(rb, &r); err != nil {
		return err
	}

	// Path must be formatted correctly if made it here
	space := strings.Split(path, parser.Delimiter)[0]

	amountDownloaded := 0
	for _, h := range r.Children {
		chunk := space + parser.Delimiter + h
		exists, b, err := cli.Resolve(chunk)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w:%s", ErrMissing, chunk)
		}
		if _, err := f.Write(b); err != nil {
			return err
		}
		size := len(b)
		color.Yellow("downloaded chunk=%s size=%dKB", chunk, float64(size)/units.KiB)
		amountDownloaded += size
	}
	color.Green("download path=%s size=%dMB", path, float64(amountDownloaded)/units.KiB)
	return nil
}

// Delete all hashes under a root
func Delete(ctx context.Context, cli client.Client, path string, priv *ecdsa.PrivateKey) error {
	exists, rb, err := cli.Resolve(path)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w:%s", ErrMissing, path)
	}
	var r Root
	if err := json.Unmarshal(rb, &r); err != nil {
		return err
	}
	// Path must be formatted correctly if made it here
	spl := strings.Split(path, parser.Delimiter)
	space := spl[0]
	root := spl[1]
	opts := []client.OpOption{client.WithPollTx()}
	totalCost := uint64(0)
	for _, h := range r.Children {
		tx := &chain.DeleteTx{
			BaseTx: &chain.BaseTx{},
			Space:  space,
			Key:    h,
		}
		txID, cost, err := client.SignIssueRawTx(ctx, cli, tx, priv, opts...)
		if err != nil {
			return err
		}
		totalCost += cost
		color.Yellow("deleted k=%s txID=%s cost=%d totalCost=%d", h, txID, cost, totalCost)
	}
	tx := &chain.DeleteTx{
		BaseTx: &chain.BaseTx{},
		Space:  space,
		Key:    root,
	}
	txID, cost, err := client.SignIssueRawTx(ctx, cli, tx, priv, opts...)
	if err != nil {
		return err
	}
	totalCost += cost
	color.Green("deleted root=%s txID=%s cost=%d totalCost=%d", path, txID, cost, totalCost)
	return nil
}
