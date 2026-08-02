//go:build !desktop

package main

import "io/fs"

// nilFS 空文件系统，无 desktop build tag 时使用。
type nilFS struct{}

func (nilFS) Open(string) (fs.File, error) { return nil, fs.ErrNotExist }

// assets 在无 desktop build tag 时为空占位，保证测试与默认构建可用。
var assets fs.FS = nilFS{}
