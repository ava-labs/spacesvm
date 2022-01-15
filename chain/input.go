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
	switch i.Typ {
	case Claim:
		return &ClaimTx{
			BaseTx: &BaseTx{},
			Space:  i.Space,
		}, nil
	case Lifeline:
		return &LifelineTx{
			BaseTx: &BaseTx{},
			Space:  i.Space,
			Units:  i.Units,
		}, nil
	case Set:
		return &SetTx{
			BaseTx: &BaseTx{},
			Space:  i.Space,
			Key:    i.Key,
			Value:  i.Value,
		}, nil
	case Delete:
		return &DeleteTx{
			BaseTx: &BaseTx{},
			Space:  i.Space,
			Key:    i.Key,
		}, nil
	case Move:
		return &MoveTx{
			BaseTx: &BaseTx{},
			Space:  i.Space,
			To:     i.To,
		}, nil
	case Transfer:
		return &TransferTx{
			BaseTx: &BaseTx{},
			To:     i.To,
			Units:  i.Units,
		}, nil
	default:
		return nil, errors.New("invalid type")
	}
}
