package tree

import (
	"errors"
)

var (
	ErrEmpty   = errors.New("file is empty")
	ErrMissing = errors.New("required file is missing")
)
