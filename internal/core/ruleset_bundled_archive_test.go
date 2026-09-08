package core

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRuleSetReadBundledSnapshot(t *testing.T) {
	t.Parallel()
	content := []byte("full:example.com\n")
	bundle := bundledTestFS(t, content)
	snapshot, err := readBundledRuleSet(bundle, "test")
	if err != nil || !bytes.Equal(content, snapshot.Content) || snapshot.UpdatedAt.Unix() != 1 {
		t.Fatalf("snapshot = %+v, error = %v", snapshot, err)
	}
}

func TestRuleSetBundleMetadataErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		manifest string
	}{
		{name: "missing manifest"},
		{name: "invalid json", manifest: "{"},
		{name: "missing tag", manifest: "{}"},
		{name: "missing time", manifest: `{"test":{"sha256":"unused"}}`},
		{name: "invalid time", manifest: `{"test":{"updated_at":"invalid"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bundle := fstest.MapFS{}
			if tt.manifest != "" {
				bundle["ruleset_bundle/manifest.json"] = &fstest.MapFile{Data: []byte(tt.manifest)}
			}
			if _, err := readBundledRuleSet(bundle, "test"); err == nil {
				t.Fatal("expected metadata error")
			}
		})
	}
}

func TestRuleSetBundleArchiveErrors(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"missing", "invalid", "truncated", "checksum"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			bundle := bundledTestFS(t, []byte("original"))
			const path = "ruleset_bundle/test.gz"
			switch name {
			case "missing":
				delete(bundle, path)
			case "invalid":
				bundle[path].Data = []byte("not gzip")
			case "truncated":
				bundle[path].Data = bundle[path].Data[:len(bundle[path].Data)-1]
			case "checksum":
				bundle[path] = bundledTestFS(t, []byte("changed"))[path]
			}
			if _, err := readBundledRuleSet(bundle, "test"); err == nil {
				t.Fatal("expected archive error")
			}
		})
	}
}

func TestRuleSetBundleDecompressionLimit(t *testing.T) {
	t.Parallel()
	bundle := bundledTestFS(t, []byte(strings.Repeat("x", maxRuleSetBodyBytes+1)))
	if _, err := readBundledRuleSet(bundle, "test"); err == nil {
		t.Fatal("expected decompression size limit")
	}
}

func bundledTestFS(t *testing.T, content []byte) fstest.MapFS {
	t.Helper()
	var archive bytes.Buffer
	writer := gzip.NewWriter(&archive)
	if _, err := writer.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf(`{"test":{"sha256":"%x","updated_at":"1970-01-01T00:00:01Z"}}`, sha256.Sum256(content))
	return fstest.MapFS{
		"ruleset_bundle/manifest.json": &fstest.MapFile{Data: []byte(manifest)},
		"ruleset_bundle/test.gz":       &fstest.MapFile{Data: archive.Bytes()},
	}
}
