// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"errors"
)

const (
	MaxPrefixSize = 256
	MaxKeyLength  = 256
)

var (
	ErrPrefixEmpty         = errors.New("prefix cannot be empty")
	ErrPrefixTooBig        = errors.New("prefix too big")
	ErrInvalidKeyDelimiter = errors.New("key has unexpected delimiters; only one sub-key is supported")
)

var (
	delimiter   = byte(0x2f) // '/'
	noPrefixEnd = []byte{0}
)

// GetPrefix returns the prefixed key and range query end key for list calls.
func ParseKey(key []byte) (pfx []byte, k []byte, end []byte, err error) {
	if len(key) == 0 {
		return nil, nil, nil, ErrPrefixEmpty
	}
	if bytes.Count(key, []byte{delimiter}) > 1 {
		return nil, nil, nil, ErrInvalidKeyDelimiter
	}

	idx := bytes.IndexRune(key, rune(delimiter))
	switch {
	case idx == -1: // "foo"
		pfx = append(key, delimiter)
	case idx == len(key)-1: // "foo/"
		pfx = key
	default: // "a/b", then "a/" becomes prefix
		splits := bytes.Split(key, []byte{delimiter})
		pfx = append(splits[0], delimiter)
		k = splits[1]
	}

	// next lexicographical key (range end) for prefix queries
	end = getRangeEnd(k)

	if len(pfx) > MaxPrefixSize {
		return nil, nil, nil, ErrPrefixTooBig
	}
	if len(key) > MaxKeyLength {
		return nil, nil, nil, ErrKeyTooBig
	}

	return pfx, k, end, nil
}

// next lexicographical key (range end) for prefix queries
func getRangeEnd(k []byte) (end []byte) {
	end = make([]byte, len(k))
	copy(end, k)
	pfxEndExist := false
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] < 0xff {
			end[i]++
			end = end[:i+1]
			pfxEndExist = true
			break
		}
	}
	if !pfxEndExist {
		// next prefix does not exist (e.g., 0xffff);
		// default to special end key
		end = noPrefixEnd
	}
	return end
}
