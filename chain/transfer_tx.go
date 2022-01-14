// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ava-labs/spacesvm/parser"
)

var _ UnsignedTransaction = &SetTx{}

type TransferTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`
	To      common.Address `serialize:"true" json:"to"`
	Value   uint64         `serialize:"true" json:"value"`
}

func (t *TransferTx) Execute(c *TransactionContext) error {
	// Note this also prevents someone from transferring a prefix to themselves.
	if t.To == c.Sender {
		return ErrNonActionable
	}

	actionable := false
	if t.Value > 0 {
		actionable = true
		if _, err := ModifyBalance(c.Database, c.Sender, false, t.Value); err != nil {
			return err
		}
		if _, err := ModifyBalance(c.Database, t.To, true, t.Value); err != nil {
			return err
		}
	}
	// TODO: move prefix to tx model outside of base
	if len(t.Prefix()) > 0 { //nolint:nestif
		if err := parser.CheckPrefix(t.Prefix()); err != nil {
			return err
		}
		actionable = true
		p, exists, err := GetPrefixInfo(c.Database, t.Prefix())
		if err != nil {
			return err
		}
		if !exists {
			return ErrPrefixMissing
		}
		if p.Owner != c.Sender {
			return ErrUnauthorized
		}
		p.Owner = t.To
		if err := PutPrefixInfo(c.Database, t.Prefix(), p, 0); err != nil { // make optional to update prefix expiry
			return err
		}
	}
	if !actionable {
		return ErrNonActionable
	}
	return nil
}

func (t *TransferTx) Copy() UnsignedTransaction {
	to := make([]byte, common.AddressLength)
	copy(to, t.To[:])
	return &TransferTx{
		BaseTx: t.BaseTx.Copy(),
		To:     common.BytesToAddress(to),
		Value:  t.Value,
	}
}
