package mygo

import (
	"context"

	"github.com/xuthus5/boxd/control"
)

// Runtime is the host-owned control-plane instance passed to the native UI.
// The adapter deliberately knows nothing about storage, kernel drivers, or the
// Web console composition.
type Runtime struct {
	Control      control.Application
	Close        func() error
	DataDir      string
	DatabasePath string
	Version      string
	Commit       string
	BuiltAt      string
}

// Loader initializes the control plane asynchronously so the native window can
// render immediately.
type Loader func(context.Context) (*Runtime, error)
