package tree

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"os"
	"strings"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
)

type Root struct {
	Children []string `json:"children"`
}

func Upload(ctx context.Context, cli client.Client, priv *ecdsa.PrivateKey, space string, f *os.File, chunkSize int) (string, error) {
	hashes := []string{}
	chunk := make([]byte, chunkSize)
	shouldExit := false
	opts := []client.OpOption{client.WithPollTx(), client.WithInfo(space)}
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
		txID, err := client.SignIssueRawTx(ctx, cli, tx, priv, opts...)
		if err != nil {
			return "", err
		}
		color.Yellow("uploaded k=%s txID=%s", k, txID)
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
	txID, err := client.SignIssueRawTx(ctx, cli, tx, priv, opts...)
	if err != nil {
		return "", err
	}
	color.Green("uploaded root=%s txID=%s", rk, txID)
	return rk, nil
}

func Download(cli *client.Client, priv *ecdsa.PrivateKey, kvs []*chain.KeyValue) error {
	return nil
}
