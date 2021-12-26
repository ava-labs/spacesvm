// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"io"

	"github.com/fatih/color"
)

var colors = []*color.Color{
	color.New(color.FgGreen),
	color.New(color.FgYellow),
	color.New(color.FgBlue),
	color.New(color.FgMagenta),
	color.New(color.FgCyan),
}

type writer struct {
	col  *color.Color
	name string
	w    io.Writer
}

func (wr *writer) Write(p []byte) (n int, err error) {
	wr.col.Fprintf(wr.w, "[%s]	", wr.name)
	return wr.w.Write(p)
}
