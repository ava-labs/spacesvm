// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"errors"
	"testing"
)

func TestParseKey(t *testing.T) {
	t.Parallel()

	tt := []struct {
		key []byte
		pfx []byte
		k   []byte
		end []byte
		err error
	}{
		{
			key: []byte("a/b/"),
			pfx: nil,
			k:   nil,
			end: nil,
			err: ErrInvalidKeyDelimiter,
		},
		{
			key: []byte("foo"),
			pfx: []byte("foo/"),
			k:   nil,
			end: noPrefixEnd,
			err: nil,
		},
		{
			key: []byte("foo/"),
			pfx: []byte("foo/"),
			k:   nil,
			end: noPrefixEnd,
			err: nil,
		},
		{
			key: []byte("a/b"),
			pfx: []byte("a/"),
			k:   []byte("b"),
			end: []byte("c"),
			err: nil,
		},
		{
			key: []byte("foo/1"),
			pfx: []byte("foo/"),
			k:   []byte("1"),
			end: []byte("2"),
			err: nil,
		},
		{
			key: []byte("foo/hello"),
			pfx: []byte("foo/"),
			k:   []byte("hello"),
			end: []byte("hellp"),
			err: nil,
		},
		{
			key: nil,
			pfx: nil,
			k:   nil,
			end: nil,
			err: ErrPrefixEmpty,
		},
		{
			key: bytes.Repeat([]byte{'a'}, MaxPrefixSize+1),
			pfx: nil,
			k:   nil,
			end: nil,
			err: ErrPrefixTooBig,
		},
		{
			key: append([]byte("foo/"), bytes.Repeat([]byte{'a'}, MaxKeyLength+1)...),
			pfx: nil,
			k:   nil,
			end: nil,
			err: ErrKeyTooBig,
		},
	}
	for i, tv := range tt {
		pfx, k, end, err := ParseKey(tv.key)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: err expected %v, got %v", i, tv.err, err)
		}
		if !bytes.Equal(pfx, tv.pfx) {
			t.Fatalf("#%d: pfx expected %q, got %q", i, tv.pfx, pfx)
		}
		if !bytes.Equal(k, tv.k) {
			t.Fatalf("#%d: k expected %q, got %q", i, tv.k, k)
		}
		if !bytes.Equal(end, tv.end) {
			t.Fatalf("#%d: end expected %q, got %q", i, tv.end, end)
		}
	}
}
