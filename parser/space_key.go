// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package parser defines storage key parsing operations.
package parser

import (
	"errors"
	"regexp"
	"strings"
)

const (
	MaxIdentifierSize      = 256
	Delimiter              = "/"
	ByteDelimiter     byte = '/'
)

var (
	ErrInvalidContents = errors.New("spaces and keys must be ^[a-z0-9]{1,256}$")
	ErrInvalidPath     = errors.New("path is not of the form space/key")

	reg *regexp.Regexp
)

func init() {
	reg = regexp.MustCompile("^[a-z0-9]{1,256}$")
}

// CheckContents returns an error if the identifier (space or key) format is invalid.
func CheckContents(identifier string) error {
	if !reg.MatchString(identifier) {
		return ErrInvalidContents
	}
	return nil
}

func ResolvePath(path string) (space string, key string, err error) {
	segments := strings.Split(path, Delimiter)
	if len(segments) != 2 {
		return "", "", ErrInvalidPath
	}
	space = segments[0]
	if err := CheckContents(space); err != nil {
		return "", "", err
	}
	key = segments[1]
	if err := CheckContents(key); err != nil {
		return "", "", err
	}
	return
}
