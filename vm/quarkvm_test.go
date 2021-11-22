// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"testing"

	"github.com/ava-labs/quarkvm/codec"
	"github.com/ava-labs/quarkvm/crypto/ed25519"
	"github.com/ava-labs/quarkvm/transaction"
)

func TestIssueTxErrInvalidSig(t *testing.T) {
	priv, err := ed25519.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.PublicKey()

	utx := transaction.Unsigned{
		PublicKey: pub.Bytes(),
		Op:        "Put",
		Key:       "foo",
		Value:     "bar",
	}
	d, err := codec.Marshal(utx)
	if err != nil {
		t.Fatal(err)
	}

	sig, err := priv.Sign(d)
	if err != nil {
		panic(err)
	}
	svc := Service{vm: &VM{}}
	if err := svc.IssueTx(
		nil,
		&IssueTxArgs{
			Transaction: transaction.New(
				transaction.Unsigned{
					PublicKey: pub.Bytes(),
					Op:        "Put",
					Key:       "foo",
					Value:     "ba0", // corrupted value, invalid signature
				},
				sig,
			),
		},
		&IssueTxReply{},
	); err != transaction.ErrInvalidSig {
		t.Fatalf("unexpected error %v, expected %v", err, transaction.ErrInvalidSig)
	}
}
