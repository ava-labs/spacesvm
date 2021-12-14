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
			key: append(maxPfx('b'), maxK('c')...),
			pfx: maxPfx('b'),
			k:   maxK('c'),
			end: append(maxK('c'), 'd')[1:],
			err: nil,
		},
		{
			key: append([]byte{'a'}, maxPfx('a')...),
			pfx: nil,
			k:   nil,
			end: nil,
			err: ErrPrefixTooBig,
		},
		{
			key: append(maxPfx('a'), append(maxK('e'), 'e')...),
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

func maxPfx(b byte) []byte {
	return append(bytes.Repeat([]byte{b}, MaxPrefixSize-1), delimiter)
}

func maxK(b byte) []byte {
	return bytes.Repeat([]byte{b}, MaxKeyLength)
}
