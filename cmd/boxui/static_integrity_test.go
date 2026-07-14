//go:build embed_ui

package main

import (
	"io/fs"
	"mime"
	"path"
	"regexp"
	"strings"
	"testing"
)

var assetReferencePattern = regexp.MustCompile(`(?:/|\"|')?(assets/[A-Za-z0-9_.-]+\.(?:js|css))`)

var relativeReferencePattern = regexp.MustCompile(`["'](\./[A-Za-z0-9_.-]+\.(?:js|css))["']`)

func TestEmbeddedUIAssetIntegrity(t *testing.T) {
	root, err := fs.Sub(embeddedStaticFS, "ui/dist")
	if err != nil {
		t.Fatal(err)
	}
	index, err := fs.ReadFile(root, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	checkAssetReferences(t, root, "index.html", index)

	foundUnderscoreAsset := false
	err = fs.WalkDir(root, "assets", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasPrefix(path.Base(name), "_") {
			foundUnderscoreAsset = true
		}
		extension := path.Ext(name)
		if extension == ".js" || extension == ".css" {
			if mime.TypeByExtension(extension) == "" {
				t.Fatalf("asset %s has no MIME type", name)
			}
		}
		if extension == ".js" {
			content, err := fs.ReadFile(root, name)
			if err != nil {
				return err
			}
			checkAssetReferences(t, root, name, content)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !foundUnderscoreAsset {
		t.Fatal("embedded UI does not contain an underscore-prefixed Vite asset")
	}
}

func checkAssetReferences(t *testing.T, root fs.FS, source string, content []byte) {
	t.Helper()
	for _, match := range assetReferencePattern.FindAllSubmatch(content, -1) {
		name := string(match[1])
		if _, err := fs.Stat(root, name); err != nil {
			t.Errorf("%s references missing asset %s: %v", source, name, err)
		}
	}
	for _, match := range relativeReferencePattern.FindAllSubmatch(content, -1) {
		name := path.Clean(path.Join(path.Dir(source), string(match[1])))
		if _, err := fs.Stat(root, name); err != nil {
			t.Errorf("%s references missing relative asset %s: %v", source, name, err)
		}
	}
}
