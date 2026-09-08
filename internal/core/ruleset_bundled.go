package core

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

//go:embed ruleset_bundle/*.gz ruleset_bundle/manifest.json
var bundledRuleSets embed.FS

type bundledRuleSetSnapshot struct {
	SHA256    string    `json:"sha256"`
	UpdatedAt time.Time `json:"updated_at"`
	Content   []byte    `json:"-"`
}

// InstallBundled 离线安装默认规则集；保留已有有效缓存，失败后可安全重试。
func (i *LoyalsoldierRuleSetInstaller) InstallBundled(ctx context.Context) ([]map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(i.ruleSetDir, 0700); err != nil {
		return nil, fmt.Errorf("create rule-set dir: %w", err)
	}
	entries := make([]map[string]any, 0, len(i.sources))
	for _, src := range i.sources {
		if err := i.installBundledRuleSet(ctx, src); err != nil {
			return nil, fmt.Errorf("install bundled %s: %w", src.Tag, err)
		}
		entries = append(entries, localRuleSetEntry(i.ruleSetDir, src))
	}
	return entries, nil
}

func (i *LoyalsoldierRuleSetInstaller) installBundledRuleSet(ctx context.Context, src RuleSetSource) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path := filepath.Join(i.ruleSetDir, src.FileName)
	valid, err := cachedRuleSetValid(ctx, path, src.Format)
	if err != nil {
		return err
	}
	if valid {
		return ctx.Err()
	}
	snapshot, err := readBundledRuleSet(bundledRuleSets, src.Tag)
	if err != nil {
		return err
	}
	snapshot.Content, err = convertRuleSetData(src, snapshot.Content)
	if err != nil {
		return err
	}
	if err := validateRuleSetData(ctx, snapshot.Content, src.Format); err != nil {
		return fmt.Errorf("validate snapshot: %w", err)
	}
	return writeBundledRuleSetSnapshot(ctx, path, snapshot)
}

func writeBundledRuleSetSnapshot(ctx context.Context, path string, snapshot bundledRuleSetSnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// 在 rename 前设置上游时间，防止初次解压把旧快照误报为刚更新。
	if err := atomicWriteRuleSetFile(path, snapshot.Content, snapshot.UpdatedAt); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}
	return nil
}

func readBundledRuleSet(bundle fs.FS, tag string) (bundledRuleSetSnapshot, error) {
	var empty bundledRuleSetSnapshot
	manifest, err := fs.ReadFile(bundle, "ruleset_bundle/manifest.json")
	if err != nil {
		return empty, fmt.Errorf("read bundle manifest: %w", err)
	}
	snapshots := make(map[string]bundledRuleSetSnapshot)
	if err := json.Unmarshal(manifest, &snapshots); err != nil {
		return empty, fmt.Errorf("decode bundle manifest: %w", err)
	}
	snapshot, ok := snapshots[tag]
	if !ok || snapshot.UpdatedAt.IsZero() {
		return empty, fmt.Errorf("missing bundle metadata for %s", tag)
	}
	archive, err := fs.ReadFile(bundle, "ruleset_bundle/"+tag+".gz")
	if err != nil {
		return empty, fmt.Errorf("read bundle archive: %w", err)
	}
	content, err := decompressRuleSetArchive(archive)
	if err != nil {
		return empty, fmt.Errorf("decompress bundle: %w", err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(content)) != snapshot.SHA256 {
		return empty, fmt.Errorf("bundle checksum mismatch for %s", tag)
	}
	snapshot.Content = content
	return snapshot, nil
}

func decompressRuleSetArchive(archive []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	return readRuleSetBody(reader)
}
