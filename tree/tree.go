// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package tree

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
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
	Contents []byte   `json:"contents"`
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
	uploaded := map[string]struct{}{}
	for !shouldExit {
		read, err := f.Read(chunk)
		if errors.Is(err, io.EOF) || read == 0 {
			break
		}
		if err != nil {
			return "", fmt.Errorf("%w: read error", err)
		}
		if read < chunkSize {
			shouldExit = true
			chunk = chunk[:read]

			// Use small file optimization
			if len(hashes) == 0 {
				break
			}
		}
		k := strings.ToLower(common.Bytes2Hex(crypto.Keccak256(chunk)))
		if _, ok := uploaded[k]; ok {
			color.Yellow("already uploaded k=%s, skipping", k)
		} else {
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
			uploaded[k] = struct{}{}
		}
		hashes = append(hashes, k)
	}

	r := &Root{}
	if len(hashes) == 0 {
		if len(chunk) == 0 {
			return "", ErrEmpty
		}
		r.Contents = chunk
	} else {
		r.Children = hashes
	}

	rb, err := json.Marshal(r)
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
	color.Yellow("uploaded root=%s txID=%s cost=%d totalCost=%d", rk, txID, cost, totalCost)
	return space + parser.Delimiter + rk, nil
}

// TODO: make multi-threaded
func Download(ctx context.Context, cli client.Client, path string, f io.Writer) error {
	exists, rb, _, err := cli.Resolve(ctx, path)
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

	// Use small file optimization
	if contentLen := len(r.Contents); contentLen > 0 {
		if _, err := f.Write(r.Contents); err != nil {
			return err
		}
		color.Yellow("downloaded path=%s size=%fKB", path, float64(contentLen)/units.KiB)
		return nil
	}

	if len(r.Children) == 0 {
		return ErrEmpty
	}

	// Path must be formatted correctly if made it here
	space := strings.Split(path, parser.Delimiter)[0]

	amountDownloaded := 0
	for _, h := range r.Children {
		chunk := space + parser.Delimiter + h
		exists, b, _, err := cli.Resolve(ctx, chunk)
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
		color.Yellow("downloaded chunk=%s size=%fKB", chunk, float64(size)/units.KiB)
		amountDownloaded += size
	}
	color.Yellow("download path=%s size=%fMB", path, float64(amountDownloaded)/units.MiB)
	return nil
}

// Delete all hashes under a root
func Delete(ctx context.Context, cli client.Client, path string, priv *ecdsa.PrivateKey) error {
	exists, rb, _, err := cli.Resolve(ctx, path)
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
	deleted := map[string]struct{}{}
	for _, h := range r.Children {
		if _, ok := deleted[h]; ok {
			color.Yellow("already deleted k=%s, skipping", h)
			continue
		}
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
		deleted[h] = struct{}{}
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
	color.Yellow("deleted root=%s txID=%s cost=%d totalCost=%d", path, txID, cost, totalCost)
	return nil
}
