package singbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/sagernet/sing-box/include"

	"github.com/xuthus5/boxd/kernel"
)

var _ kernel.DefaultsFeature = (*Engine)(nil)

// InstallDefaults implements kernel.DefaultsFeature. The sing-box driver owns
// the cache-file default because it is an engine-specific path; callers can
// request a specific section or all supported defaults with an empty section.
func (e *Engine) InstallDefaults(ctx context.Context, document kernel.ConfigDocument, section string) (kernel.ConfigDocument, any, error) {
	if err := ctx.Err(); err != nil {
		return kernel.ConfigDocument{}, nil, err
	}
	if section != "" && section != "cache" {
		return kernel.ConfigDocument{}, nil, kernel.NewError(
			kernel.ErrCodeInvalidConfig,
			"driver.singbox.defaults",
			"unsupported defaults section "+section,
		).WithHint("supported sections: cache")
	}
	parsed, err := parseOptions(ctx, document, e.workDir)
	if err != nil {
		return kernel.ConfigDocument{}, nil, kernel.WrapError(kernel.ErrCodeInvalidConfig, "driver.singbox.defaults", "parse configuration for defaults", err)
	}
	body, err := parsed.MarshalJSONContext(include.Context(ctx))
	if err != nil {
		return kernel.ConfigDocument{}, nil, kernel.WrapError(kernel.ErrCodeInternal, "driver.singbox.defaults", "marshal configuration defaults", err)
	}
	digest := sha256.Sum256(body)
	result := kernel.ConfigDocument{
		Format:   "json",
		Body:     body,
		Revision: "config:" + hex.EncodeToString(digest[:]),
	}
	cachePath := ""
	if parsed.Experimental != nil && parsed.Experimental.CacheFile != nil {
		cachePath = parsed.Experimental.CacheFile.Path
	}
	return result, map[string]any{"section": "cache", "cache_path": cachePath}, nil
}
