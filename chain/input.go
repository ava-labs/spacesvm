package chain

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

type Input struct {
	Typ   string         `json:"type"`
	Space string         `json:"space"`
	Key   string         `json:"key"`
	Value []byte         `json:"value"`
	To    common.Address `json:"to"`
	Units uint64         `json:"units"`
}

func (i *Input) Decode() (UnsignedTransaction, error) {
	// TODO: add other inputs
	switch i.Typ {
	case "ClaimTx":
		return &ClaimTx{
			BaseTx: &BaseTx{},
			Space:  i.Space,
		}, nil
	default:
		return nil, errors.New("invalid type")
	}
}
