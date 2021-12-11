// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"testing"
)

func TestPrefixValueKey(t *testing.T) {
	t.Parallel()

	tt := []struct {
		pfx      []byte
		key      []byte
		valueKey []byte
	}{
		{
			pfx:      []byte("foo"),
			key:      []byte("hello"),
			valueKey: append([]byte{keyPrefix}, []byte("/foo/hello")...),
		},
		{
			pfx:      []byte("foo/"),
			key:      []byte("hello"),
			valueKey: append([]byte{keyPrefix}, []byte("/foo/hello")...),
		},
	}
	for i, tv := range tt {
		vv := PrefixValueKey(tv.pfx, tv.key)
		if !bytes.Equal(tv.valueKey, vv) {
			t.Fatalf("#%d: value expected %q, got %q", i, tv.valueKey, vv)
		}
	}
}
