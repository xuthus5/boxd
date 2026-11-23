package singbox

import (
	"bytes"
	"context"
	"testing"

	"github.com/xuthus5/boxd/kernel"
)

func TestInstallDefaultsNormalizesCachePath(t *testing.T) {
	ctx := context.Background()
	workDir := t.TempDir()
	document := kernel.ConfigDocument{Format: "json", Body: []byte(`{}`)}
	parsed, err := parseOptions(ctx, document, workDir)
	if err != nil {
		t.Fatalf("parseOptions() error = %v", err)
	}
	engine := newEngine(kernelOpenOptions(workDir, document), parsed)
	result, summary, err := engine.InstallDefaults(ctx, document, "cache")
	if err != nil {
		t.Fatalf("InstallDefaults() error = %v", err)
	}
	if result.Format != "json" || result.Revision == "" || !bytes.Contains(result.Body, []byte(workDir)) {
		t.Fatalf("InstallDefaults() = %+v", result)
	}
	cachePath, _ := summary.(map[string]any)["cache_path"].(string)
	if cachePath == "" || !bytes.Contains(result.Body, []byte(cachePath)) {
		t.Fatalf("defaults summary = %+v", summary)
	}
	if _, _, err := engine.InstallDefaults(ctx, document, "unknown"); err == nil {
		t.Fatal("InstallDefaults() accepted an unsupported section")
	}
}
