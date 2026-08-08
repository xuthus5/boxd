package core

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

const (
	ruleSetAutoUpdateKey       = "ruleset_auto_update"
	defaultRuleSetAutoInterval = "24h"
	singBoxRuleSetBucket       = "rule_set"
)

var (
	ErrRuleSetNotFound      = errors.New("rule-set not found in config")
	ErrRuleSetNotUpdatable  = errors.New("rule-set is not updatable")
	ErrRuleSetCacheDisabled = errors.New("rule-set cache is unavailable")
)

// RuleSetUpdateRequest 选择要更新的规则集。
// Tags 为空表示更新全部可更新项；Types 可过滤 local/remote。
type RuleSetUpdateRequest struct {
	Tags  []string `json:"tags"`
	Types []string `json:"types"`
}

type RuleSetUpdater struct {
	configPath string
	cachePath  string
	installer  *LoyalsoldierRuleSetInstaller
	client     *http.Client
	stop       func() error
	start      func() error
}

func NewRuleSetUpdater(configPath, dataDir string, installer *LoyalsoldierRuleSetInstaller, stop, start func() error) *RuleSetUpdater {
	if installer == nil {
		installer = NewLoyalsoldierRuleSetInstaller(dataDir)
	}
	return &RuleSetUpdater{
		configPath: configPath,
		cachePath:  filepath.Join(dataDir, "cache.db"),
		installer:  installer,
		client:     newPublicHTTPClient(ruleSetHTTPTimeout),
		stop:       stop,
		start:      start,
	}
}

func DefaultRuleSetAutoUpdate() model.RuleSetAutoUpdate {
	return model.RuleSetAutoUpdate{Enabled: false, Interval: defaultRuleSetAutoInterval}
}

func ValidateRuleSetAutoUpdate(cfg model.RuleSetAutoUpdate) error {
	if strings.TrimSpace(cfg.Interval) == "" {
		return fmt.Errorf("interval is required")
	}
	d, err := time.ParseDuration(cfg.Interval)
	if err != nil || d <= 0 {
		return fmt.Errorf("interval must be a positive duration such as 24h")
	}
	return nil
}

func (m *SettingsManager) RuleSetAutoUpdate() (model.RuleSetAutoUpdate, error) {
	value := m.Get(ruleSetAutoUpdateKey)
	if value == "" {
		return DefaultRuleSetAutoUpdate(), nil
	}
	var cfg model.RuleSetAutoUpdate
	if err := json.Unmarshal([]byte(value), &cfg); err != nil {
		return model.RuleSetAutoUpdate{}, fmt.Errorf("decoding ruleset auto update: %w", err)
	}
	if cfg.Interval == "" {
		cfg.Interval = defaultRuleSetAutoInterval
	}
	return cfg, nil
}

func (m *SettingsManager) SetRuleSetAutoUpdate(cfg model.RuleSetAutoUpdate) error {
	if err := ValidateRuleSetAutoUpdate(cfg); err != nil {
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding ruleset auto update: %w", err)
	}
	return m.Set(ruleSetAutoUpdateKey, string(data))
}

func (u *RuleSetUpdater) Status(ctx context.Context) ([]model.RuleSetStatusItem, error) {
	_ = ctx
	cfg, err := u.loadConfig()
	if err != nil {
		return nil, err
	}
	entries := ruleSetEntries(cfg)
	cache, _ := u.openCacheReadOnly()
	if cache != nil {
		defer func() { _ = cache.Close() }()
	}
	items := make([]model.RuleSetStatusItem, 0, len(entries))
	for _, entry := range entries {
		items = append(items, u.statusOf(entry, cache))
	}
	return items, nil
}

func (u *RuleSetUpdater) Update(ctx context.Context, req RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error) {
	cfg, err := u.loadConfig()
	if err != nil {
		return model.RuleSetUpdateResponse{}, err
	}
	entries := ruleSetEntries(cfg)
	selected := selectRuleSets(entries, req)
	resp := model.RuleSetUpdateResponse{Results: make([]model.RuleSetUpdateResult, 0, len(selected))}
	needsCacheWrite := false
	for _, entry := range selected {
		if stringValue(entry["type"]) != "remote" {
			continue
		}
		if ValidatePublicHTTPURL(stringValue(entry["url"])) == nil {
			needsCacheWrite = true
			break
		}
	}
	stopped := false
	if needsCacheWrite && u.stop != nil {
		if err := u.stop(); err != nil {
			return model.RuleSetUpdateResponse{}, fmt.Errorf("stop kernel before rule-set update: %w", err)
		}
		stopped = true
	}
	for _, entry := range selected {
		result := u.updateOne(ctx, entry)
		resp.Results = append(resp.Results, result)
		switch {
		case result.OK && !result.NotModified:
			resp.UpdatedCount++
		case result.OK && result.NotModified:
		case result.ErrorCode == RuleSetErrorNotUpdatable || result.ErrorCode == RuleSetErrorUnsupported ||
			strings.Contains(result.Error, "not auto-updated") || strings.Contains(result.Error, "not updatable"):
			resp.SkippedCount++
		default:
			resp.FailedCount++
		}
	}
	if stopped && u.start != nil {
		if err := u.start(); err != nil {
			return resp, fmt.Errorf("start kernel after rule-set update: %w", err)
		}
		resp.Restarted = true
	}
	return resp, nil
}

func (u *RuleSetUpdater) updateOne(ctx context.Context, entry map[string]any) model.RuleSetUpdateResult {
	tag, _ := entry["tag"].(string)
	typ, _ := entry["type"].(string)
	if typ == "" {
		typ = "inline"
	}
	result := model.RuleSetUpdateResult{Tag: tag, Type: typ}
	switch typ {
	case "local":
		return u.updateLocal(ctx, entry, result)
	case "remote":
		return u.updateRemote(ctx, entry, result)
	default:
		return failRuleSetResult(result, ErrRuleSetNotUpdatable.Error(), ErrRuleSetNotUpdatable)
	}
}

func (u *RuleSetUpdater) updateLocal(ctx context.Context, entry map[string]any, result model.RuleSetUpdateResult) model.RuleSetUpdateResult {
	tag := result.Tag
	src, ok := u.installer.SourceByTag(tag)
	if !ok {
		return failRuleSetResult(result, "custom local rule-set files are not auto-updated", nil)
	}
	path, _ := entry["path"].(string)
	if path == "" {
		path = filepath.Join(u.installer.ruleSetDir, src.FileName)
	}
	now := time.Now()
	switch src.Format {
	case "binary":
		// 内置二进制规则集（.srs）由 boxd 竞速下载后覆盖本地缓存文件，内核引用不变。
		content, err := u.installer.raceDownloadContent(ctx, src.Tag, src.URL)
		if err != nil {
			return failRuleSetResult(result, err.Error(), err)
		}
		if err := atomicWriteFile0600(path, content); err != nil {
			return failRuleSetResult(result, err.Error(), err)
		}
	default:
		ruleFile, err := u.installer.fetchAndConvert(ctx, src)
		if err != nil {
			return failRuleSetResult(result, err.Error(), err)
		}
		data, err := json.MarshalIndent(ruleFile, "", "  ")
		if err != nil {
			return failRuleSetResult(result, err.Error(), err)
		}
		if err := atomicWriteFile0600(path, data); err != nil {
			return failRuleSetResult(result, err.Error(), err)
		}
	}
	result.OK = true
	result.UpdatedAt = &now
	return result
}

func (u *RuleSetUpdater) updateRemote(ctx context.Context, entry map[string]any, result model.RuleSetUpdateResult) model.RuleSetUpdateResult {
	url, _ := entry["url"].(string)
	if strings.TrimSpace(url) == "" {
		return failRuleSetResult(result, "remote rule-set url is empty", nil)
	}
	if err := ValidatePublicHTTPURL(url); err != nil {
		return failRuleSetResult(result, err.Error(), err)
	}
	etag := ""
	if cache, err := u.openCacheReadOnly(); err == nil && cache != nil {
		if saved := loadRuleSetCache(cache, result.Tag); saved != nil {
			etag = saved.LastEtag
		}
		_ = cache.Close()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return failRuleSetResult(result, err.Error(), err)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	client := u.client
	if client == nil {
		client = newPublicHTTPClient(ruleSetHTTPTimeout)
	}
	resp, err := client.Do(req)
	if err != nil {
		return failRuleSetResult(result, err.Error(), err)
	}
	if resp == nil || resp.Body == nil {
		return failRuleSetResult(result, "rule-set response body is nil", nil)
	}
	defer func() { _ = resp.Body.Close() }()
	now := time.Now()
	switch resp.StatusCode {
	case http.StatusNotModified:
		if err := u.touchRemoteCache(result.Tag, now); err != nil && !errors.Is(err, ErrRuleSetCacheDisabled) {
			return failRuleSetResult(result, err.Error(), err)
		}
		result.OK = true
		result.NotModified = true
		result.UpdatedAt = &now
		return result
	case http.StatusOK:
	default:
		return failRuleSetResult(result, fmt.Sprintf("unexpected status %d", resp.StatusCode), nil)
	}
	if resp.ContentLength > maxRuleSetBodyBytes {
		return failRuleSetResult(result, ErrRuleSetContentTooLarge.Error(), ErrRuleSetContentTooLarge)
	}
	content, err := readRuleSetBody(resp.Body)
	if err != nil {
		return failRuleSetResult(result, err.Error(), err)
	}
	if len(content) == 0 {
		return failRuleSetResult(result, "empty rule-set body", nil)
	}
	newEtag := resp.Header.Get("Etag")
	if err := u.saveRemoteCache(result.Tag, content, newEtag, now); err != nil {
		return failRuleSetResult(result, err.Error(), err)
	}
	result.OK = true
	result.UpdatedAt = &now
	return result
}

func (u *RuleSetUpdater) statusOf(entry map[string]any, cache *bbolt.DB) model.RuleSetStatusItem {
	tag, _ := entry["tag"].(string)
	typ, _ := entry["type"].(string)
	if typ == "" {
		typ = "inline"
	}
	item := model.RuleSetStatusItem{
		Tag:            tag,
		Type:           typ,
		Format:         stringValue(entry["format"]),
		Path:           stringValue(entry["path"]),
		URL:            stringValue(entry["url"]),
		UpdateInterval: stringValue(entry["update_interval"]),
		DownloadDetour: stringValue(entry["download_detour"]),
	}
	switch typ {
	case "local":
		item.Builtin = u.installer.IsBuiltinLocal(tag)
		item.Updatable = item.Builtin
		if !item.Updatable {
			item.Note = "custom local rule-set is managed by file path"
		}
		if item.Path != "" {
			if info, err := os.Stat(item.Path); err == nil {
				t := info.ModTime()
				item.LastUpdated = &t
				item.FileSize = info.Size()
			}
		}
	case "remote":
		// 内置规则集已全部本地化（boxd 预先下载缓存），此处仅剩用户自定义 remote。
		item.Builtin = false
		item.Updatable = item.URL != ""
		if item.URL != "" {
			if err := ValidatePublicHTTPURL(item.URL); err != nil {
				item.Updatable = false
				item.Note = "remote rule-set URL is invalid or blocked"
			}
		}
		if item.UpdateInterval == "" {
			item.UpdateInterval = DefaultRemoteRuleSetInterval()
		}
		if cache != nil {
			if saved := loadRuleSetCache(cache, tag); saved != nil {
				t := saved.LastUpdated
				item.LastUpdated = &t
				item.LastEtag = saved.LastEtag
				item.FileSize = int64(len(saved.Content))
			}
		}
	default:
		item.Updatable = false
		item.Note = "inline rule-set has no remote update"
	}
	return item
}

func (u *RuleSetUpdater) loadConfig() (map[string]any, error) {
	data, err := os.ReadFile(u.configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return cfg, nil
}

func ruleSetEntries(cfg map[string]any) []map[string]any {
	route, _ := cfg["route"].(map[string]any)
	if route == nil {
		return nil
	}
	raw, _ := route["rule_set"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func selectRuleSets(entries []map[string]any, req RuleSetUpdateRequest) []map[string]any {
	tagFilter := map[string]struct{}{}
	for _, tag := range req.Tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tagFilter[tag] = struct{}{}
		}
	}
	typeFilter := map[string]struct{}{}
	for _, typ := range req.Types {
		typ = strings.ToLower(strings.TrimSpace(typ))
		if typ != "" {
			typeFilter[typ] = struct{}{}
		}
	}
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		tag, _ := entry["tag"].(string)
		typ, _ := entry["type"].(string)
		if typ == "" {
			typ = "inline"
		}
		if len(tagFilter) > 0 {
			if _, ok := tagFilter[tag]; !ok {
				continue
			}
		}
		if len(typeFilter) > 0 {
			if _, ok := typeFilter[typ]; !ok {
				continue
			}
		}
		out = append(out, entry)
	}
	return out
}

type savedRuleSetBinary struct {
	Content     []byte
	LastUpdated time.Time
	LastEtag    string
}

func (s *savedRuleSetBinary) MarshalBinary() ([]byte, error) {
	if len(s.Content) > maxRuleSetBodyBytes {
		return nil, ErrRuleSetContentTooLarge
	}
	if len(s.LastEtag) > maxRuleSetEtagBytes {
		return nil, errors.New("rule-set etag is too large")
	}
	var buffer bytes.Buffer
	if err := binary.Write(&buffer, binary.BigEndian, uint8(1)); err != nil {
		return nil, err
	}
	if err := writeUvarint(&buffer, uint64(len(s.Content))); err != nil {
		return nil, err
	}
	if _, err := buffer.Write(s.Content); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.BigEndian, s.LastUpdated.Unix()); err != nil {
		return nil, err
	}
	if err := writeUvarint(&buffer, uint64(len(s.LastEtag))); err != nil {
		return nil, err
	}
	if _, err := buffer.WriteString(s.LastEtag); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (s *savedRuleSetBinary) UnmarshalBinary(data []byte) error {
	reader := bytes.NewReader(data)
	var version uint8
	if err := binary.Read(reader, binary.BigEndian, &version); err != nil {
		return err
	}
	if version != 1 {
		return fmt.Errorf("unsupported rule-set cache version %d", version)
	}
	contentLength, err := readUvarint(reader)
	if err != nil {
		return err
	}
	if contentLength > maxRuleSetBodyBytes || contentLength > uint64(reader.Len()) {
		if contentLength > maxRuleSetBodyBytes {
			return ErrRuleSetContentTooLarge
		}
		return io.ErrUnexpectedEOF
	}
	s.Content = make([]byte, contentLength)
	if _, err := io.ReadFull(reader, s.Content); err != nil {
		return err
	}
	var lastUpdated int64
	if err := binary.Read(reader, binary.BigEndian, &lastUpdated); err != nil {
		return err
	}
	s.LastUpdated = time.Unix(lastUpdated, 0)
	etagLength, err := readUvarint(reader)
	if err != nil {
		return err
	}
	if etagLength > maxRuleSetEtagBytes || etagLength > uint64(reader.Len()) {
		if etagLength > maxRuleSetEtagBytes {
			return errors.New("rule-set etag is too large")
		}
		return io.ErrUnexpectedEOF
	}
	etagBytes := make([]byte, etagLength)
	if _, err := io.ReadFull(reader, etagBytes); err != nil {
		return err
	}
	s.LastEtag = string(etagBytes)
	return nil
}

func writeUvarint(w io.Writer, value uint64) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], value)
	_, err := w.Write(buf[:n])
	return err
}

func readUvarint(r io.ByteReader) (uint64, error) {
	return binary.ReadUvarint(r)
}

func (u *RuleSetUpdater) openCacheReadOnly() (*bbolt.DB, error) {
	if _, err := os.Stat(u.cachePath); err != nil {
		return nil, err
	}
	return bbolt.Open(u.cachePath, 0600, &bbolt.Options{Timeout: time.Second, ReadOnly: true})
}

func (u *RuleSetUpdater) openCacheReadWrite() (*bbolt.DB, error) {
	if err := os.MkdirAll(filepath.Dir(u.cachePath), 0700); err != nil {
		return nil, err
	}
	return bbolt.Open(u.cachePath, 0600, &bbolt.Options{Timeout: 2 * time.Second})
}

func loadRuleSetCache(db *bbolt.DB, tag string) *savedRuleSetBinary {
	var saved savedRuleSetBinary
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(singBoxRuleSetBucket))
		if bucket == nil {
			return os.ErrNotExist
		}
		data := bucket.Get([]byte(tag))
		if len(data) == 0 {
			return os.ErrNotExist
		}
		return saved.UnmarshalBinary(data)
	})
	if err != nil {
		return nil
	}
	return &saved
}

func (u *RuleSetUpdater) saveRemoteCache(tag string, content []byte, etag string, updated time.Time) error {
	db, err := u.openCacheReadWrite()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRuleSetCacheDisabled, err)
	}
	defer func() { _ = db.Close() }()
	saved := &savedRuleSetBinary{Content: content, LastUpdated: updated, LastEtag: etag}
	payload, err := saved.MarshalBinary()
	if err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(singBoxRuleSetBucket))
		if err != nil {
			return err
		}
		return bucket.Put([]byte(tag), payload)
	})
}

func (u *RuleSetUpdater) touchRemoteCache(tag string, updated time.Time) error {
	db, err := u.openCacheReadWrite()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRuleSetCacheDisabled, err)
	}
	defer func() { _ = db.Close() }()
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(singBoxRuleSetBucket))
		if bucket == nil {
			return nil
		}
		data := bucket.Get([]byte(tag))
		if len(data) == 0 {
			return nil
		}
		var saved savedRuleSetBinary
		if err := saved.UnmarshalBinary(data); err != nil {
			return err
		}
		saved.LastUpdated = updated
		payload, err := saved.MarshalBinary()
		if err != nil {
			return err
		}
		return bucket.Put([]byte(tag), payload)
	})
}
