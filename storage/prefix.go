package storage

import (
	"bytes"
	"errors"
)

var (
	delimiter   = byte(0x2f) // '/'
	noPrefixEnd = []byte{0}
)

var (
	ErrInvalidKeyDelimiter = errors.New("key has unexpected delimiters; only flat key or sub-key is supported")
)

// getPrefix returns the prefixed key and range query end key for list calls.
// put "foo" becomes "foo/" for its own namespace, and range ends with "foo0"
// put "fop" becomes "fop/" for its own namespace, and range ends with "fop0"
// put "foo1" becomes "foo1/" for its own namespace, and range ends with "foo10"
// For now, the storage itself does not implement key hierarchy, just flat prefix namespace.
func getPrefix(key []byte) (pfx []byte, end []byte, err error) {
	if bytes.Count(key, []byte{delimiter}) > 1 {
		return nil, nil, ErrInvalidKeyDelimiter
	}

	idx := bytes.IndexRune(key, rune(delimiter))
	switch {
	case idx == -1: // "foo"
		pfx = append(key, delimiter)
	case idx == len(key)-1: // "foo/"
		pfx = key
	default: // "a/b", then "a/" becomes prefix
		pfx = append(bytes.Split(key, []byte{delimiter})[0], delimiter)
	}

	// next lexicographical key (range end) for prefix queries
	end = make([]byte, len(pfx))
	copy(end, pfx)
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] < 0xff {
			end[i] = end[i] + 1
			end = end[:i+1]
			return pfx, end, nil
		}
	}
	// next prefix does not exist (e.g., 0xffff);
	// default to special end key
	return pfx, noPrefixEnd, nil
}
