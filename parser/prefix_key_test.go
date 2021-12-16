// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package parser

import (
	"bytes"
	"errors"
	"testing"
)

func TestCheckPrefix(t *testing.T) {
	t.Parallel()

	tt := []struct {
		pfx []byte
		err error
	}{
		{
			pfx: []byte("foo"),
			err: nil,
		},
		{
			pfx: nil,
			err: ErrPrefixEmpty,
		},
		{
			pfx: bytes.Repeat([]byte{'a'}, MaxPrefixSize+1),
			err: ErrPrefixTooBig,
		},
		{
			pfx: []byte("foo/"),
			err: ErrInvalidDelimiter,
		},
	}
	for i, tv := range tt {
		err := CheckPrefix(tv.pfx)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: err expected %v, got %v", i, tv.err, err)
		}
	}
}

func TestCheckKey(t *testing.T) {
	t.Parallel()

	tt := []struct {
		key []byte
		err error
	}{
		{
			key: []byte("foo"),
			err: nil,
		},
		{
			key: nil,
			err: ErrKeyEmpty,
		},
		{
			key: bytes.Repeat([]byte{'a'}, MaxKeySize+1),
			err: ErrKeyTooBig,
		},
		{
			key: []byte("foo/"),
			err: ErrInvalidDelimiter,
		},
	}
	for i, tv := range tt {
		err := CheckKey(tv.key)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: err expected %v, got %v", i, tv.err, err)
		}
	}
}

func TestParsePrefixKey(t *testing.T) {
	t.Parallel()

	tt := []struct {
		s   []byte
		pfx []byte
		k   []byte
		end []byte
		err error
	}{
		{
			s:   []byte("foo"),
			pfx: []byte("foo/"),
			k:   nil,
			end: noPrefixEnd,
			err: nil,
		},
		{
			s:   []byte("foo/"),
			pfx: []byte("foo/"),
			k:   nil,
			end: noPrefixEnd,
			err: nil,
		},
		{
			s:   []byte("a/b"),
			pfx: []byte("a/"),
			k:   []byte("b"),
			end: []byte("c"),
			err: nil,
		},
		{
			s:   []byte("foo/1"),
			pfx: []byte("foo/"),
			k:   []byte("1"),
			end: []byte("2"),
			err: nil,
		},
		{
			s:   []byte("foo/hello"),
			pfx: []byte("foo/"),
			k:   []byte("hello"),
			end: []byte("hellp"),
			err: nil,
		},
	}
	for i, tv := range tt {
		pfx, k, end, err := ParsePrefixKey(tv.s)
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
