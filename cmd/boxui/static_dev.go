//go:build !embed_ui

package main

import (
	"io/fs"
)

type emptyStaticFS struct{}

func (emptyStaticFS) Open(name string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

var staticFS fs.FS = emptyStaticFS{}
