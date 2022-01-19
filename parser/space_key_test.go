// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package parser

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckContents(t *testing.T) {
	t.Parallel()

	tt := []struct {
		identifier string
		err        error
	}{
		{
			identifier: "foo",
			err:        nil,
		},
		{
			identifier: "asjdkajdklajsdklajslkd27137912kskdfoo",
			err:        nil,
		},
		{
			identifier: "0xasjdkajdklajsdklajslkd27137912kskdfoo",
			err:        nil,
		},
		{
			identifier: "",
			err:        ErrInvalidContents,
		},
		{
			identifier: "Ab1",
			err:        ErrInvalidContents,
		},
		{
			identifier: "ab.1",
			err:        ErrInvalidContents,
		},
		{
			identifier: "a a",
			err:        ErrInvalidContents,
		},
		{
			identifier: "a/a",
			err:        ErrInvalidContents,
		},
		{
			identifier: "😀",
			err:        ErrInvalidContents,
		},
		{
			identifier: strings.Repeat("a", MaxIdentifierSize+1),
			err:        ErrInvalidContents,
		},
	}
	for i, tv := range tt {
		err := CheckContents(tv.identifier)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: err expected %v, got %v", i, tv.err, err)
		}
	}
}

func TestResolvePath(t *testing.T) {
	t.Parallel()

	tt := []struct {
		path string
		err  error

		space string
		key   string
	}{
		{
			path:  "foo/bar",
			err:   nil,
			space: "foo",
			key:   "bar",
		},
		{
			path:  "foo1287391723981273891723981739817398192/barasjdkajdlasjdl",
			err:   nil,
			space: "foo1287391723981273891723981739817398192",
			key:   "barasjdkajdlasjdl",
		},
		{
			path: "",
			err:  ErrInvalidPath,
		},
		{
			path: "foo/",
			err:  ErrInvalidContents,
		},
		{
			path: "foo///",
			err:  ErrInvalidPath,
		},
		{
			path: "/test",
			err:  ErrInvalidContents,
		},
		{
			path: "aajsdklasd82u8931H/bar",
			err:  ErrInvalidContents,
		},
	}
	for i, tv := range tt {
		space, key, err := ResolvePath(tv.path)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: err expected %v, got %v", i, tv.err, err)
		}
		if tv.err != nil {
			continue
		}
		if space != tv.space {
			t.Fatalf("#%d: expected %v, got %v", i, tv.space, space)
		}
		if key != tv.key {
			t.Fatalf("#%d: expected %v, got %v", i, tv.key, key)
		}
	}
}
