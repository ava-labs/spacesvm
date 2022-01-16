package tree

import (
	"crypto/ecdsa"
	"encoding/json"
	"strings"

	"github.com/ava-labs/spacesvm/chain"
	"github.com/ava-labs/spacesvm/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Root struct {
	Children []common.Hash `json:"children"`
}

// TODO: upload in place so don't need to load into memory
func Create(b []byte, chunkSize int) (string, []*chain.KeyValue, error) {
	requiredChunks := len(b)/chunkSize + 1
	max := len(b) - 1

	r := &Root{Children: make([]common.Hash, requiredChunks)}
	kv := make([]*chain.KeyValue, requiredChunks+1)
	for i := 0; i < requiredChunks; i++ {
		start := i * chunkSize
		end := (i + 1) * chunkSize
		if end > max {
			end = max
		}
		chunk := b[start:max]
		kv[i] = &chain.KeyValue{
			Key:   strings.ToLower(common.Bytes2Hex(crypto.Keccak256(chunk))),
			Value: chunk,
		}
	}

	rb, err := json.Marshal(r)
	if err != nil {
		return "", nil, err
	}
	rk := strings.ToLower(common.Bytes2Hex(crypto.Keccak256(rb)))
	kv[requiredChunks+1] = &chain.KeyValue{
		Key:   rk,
		Value: rb,
	}
	return rk, kv, nil
}

func Download(cli *client.Client, priv *ecdsa.PrivateKey, kvs []*chain.KeyValue) error {
	return nil
}
