//go:build !embed_ui

package main

import (
	"errors"
	"io/fs"
	"testing"
)

func TestEmptyStaticFSOpen(t *testing.T) {
	_, err := emptyStaticFS{}.Open("index.html")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("error = %v, want fs.ErrNotExist", err)
	}
}
