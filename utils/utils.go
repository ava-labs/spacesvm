package utils

import (
	"errors"
)

func VerifyPrefixKey(prefix []byte) error {
	if len(prefix) == 0 {
		return errors.New("prefix cannot be empty")
	}
	if len(prefix) > maxPrefixSize {
		return errors.New("prefix too big")
	}
	if bytes.IndexRune(prefix, delimiter) != -1 {
		return errors.New("prefix contains delimiter")
	}
	return nil
}
