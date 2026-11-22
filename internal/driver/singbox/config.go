package singbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"

	"github.com/xuthus5/boxd/kernel"
)

const cacheFileName = "cache.db"

func parseOptions(ctx context.Context, document kernel.ConfigDocument, workDir string) (option.Options, error) {
	if err := ctx.Err(); err != nil {
		return option.Options{}, err
	}
	if document.Format != "" && document.Format != "json" {
		return option.Options{}, fmt.Errorf("unsupported configuration format %q", document.Format)
	}
	body := document.Body
	if len(bytes.TrimSpace(body)) == 0 {
		body = []byte("{}")
	}

	parseContext := include.Context(ctx)
	var options option.Options
	if err := options.UnmarshalJSONContext(parseContext, body); err != nil {
		return option.Options{}, fmt.Errorf("parse sing-box JSON: %w", err)
	}
	if err := normalizeCachePath(&options, workDir); err != nil {
		return option.Options{}, err
	}
	return options, nil
}

func normalizeCachePath(options *option.Options, workDir string) error {
	if workDir == "" {
		current, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve working directory: %w", err)
		}
		workDir = current
	}
	absoluteWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("resolve engine working directory: %w", err)
	}
	if options.Experimental == nil {
		options.Experimental = &option.ExperimentalOptions{}
	}
	if options.Experimental.CacheFile == nil {
		options.Experimental.CacheFile = &option.CacheFileOptions{}
	}
	cachePath := options.Experimental.CacheFile.Path
	if cachePath == "" {
		cachePath = cacheFileName
	}
	if !filepath.IsAbs(cachePath) {
		cachePath = filepath.Join(absoluteWorkDir, cachePath)
	}
	options.Experimental.CacheFile.Path = cachePath
	return nil
}
