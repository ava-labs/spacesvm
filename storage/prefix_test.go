package storage

import (
	"bytes"
	"testing"
)

func TestGetPrefix(t *testing.T) {
	tt := []struct {
		key []byte
		pfx []byte
		end []byte
		err error
	}{
		{
			key: []byte("foo"),
			pfx: []byte("foo/"),
			end: []byte("foo0"),
			err: nil,
		},
		{
			key: []byte("foo/"),
			pfx: []byte("foo/"),
			end: []byte("foo0"),
			err: nil,
		},
		{
			key: []byte("a/b"),
			pfx: []byte("a/"),
			end: []byte("a/c"),
			err: nil,
		},
		{
			key: []byte("foo/1"),
			pfx: []byte("foo/"),
			end: []byte("foo/2"),
			err: nil,
		},
		{
			key: []byte("a/b/"),
			pfx: nil,
			end: nil,
			err: ErrInvalidKeyDelimiter,
		},
	}
	for i, tv := range tt {
		pfx, end, err := GetPrefix(tv.key)
		if !bytes.Equal(pfx, tv.pfx) {
			t.Fatalf("#%d: pfx expected %q, got %q", i, string(tv.pfx), string(pfx))
		}
		if !bytes.Equal(end, tv.end) {
			t.Fatalf("#%d: end expected %q, got %q", i, string(tv.end), string(end))
		}
		if err != tv.err {
			t.Fatalf("#%d: err expected %v, got %v", i, tv.err, err)
		}
	}
}
