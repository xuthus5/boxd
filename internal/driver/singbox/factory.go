// Package singbox is the only package allowed to depend on sing-box.
package singbox

import (
	"context"

	"github.com/xuthus5/boxd/kernel"
)

// Factory is the composition seam for the sing-box driver. All sing-box
// imports stay below this boundary.
type Factory struct{}

var _ kernel.Factory = (*Factory)(nil)

// NewFactory constructs the sing-box engine factory.
func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) Kind() kernel.EngineKind {
	return kernel.EngineSingBox
}

func (f *Factory) Open(ctx context.Context, options kernel.OpenOptions) (kernel.Engine, error) {
	if err := CheckCompatibility(); err != nil {
		return nil, err
	}
	parsed, err := parseOptions(ctx, options.Config, options.WorkDir)
	if err != nil {
		return nil, kernel.WrapError(
			kernel.ErrCodeInvalidConfig,
			"driver.singbox.open",
			"parse sing-box configuration",
			err,
		)
	}
	return newEngine(options, parsed), nil
}
