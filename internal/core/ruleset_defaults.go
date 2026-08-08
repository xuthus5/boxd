package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	C "github.com/sagernet/sing-box/constant"
)

type RuleSetDefaultsInstaller interface {
	Install(ctx context.Context) ([]map[string]any, error)
}

type RuleSetSource struct {
	Tag      string
	FileName string
	URL      string
}

type LoyalsoldierRuleSetInstaller struct {
	ruleSetDir string
	client     *http.Client
	sources    []RuleSetSource
}

func NewLoyalsoldierRuleSetInstaller(dataDir string) *LoyalsoldierRuleSetInstaller {
	return &LoyalsoldierRuleSetInstaller{
		ruleSetDir: filepath.Join(dataDir, "rule-sets"),
		client:     newPublicHTTPClient(ruleSetInstallerHTTPTimeout),
		sources: []RuleSetSource{
			{
				Tag:      "loyalsoldier-direct",
				FileName: "loyalsoldier-direct.json",
				URL:      "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt",
			},
			{
				Tag:      "loyalsoldier-proxy",
				FileName: "loyalsoldier-proxy.json",
				URL:      "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/proxy-list.txt",
			},
			{
				Tag:      "loyalsoldier-reject",
				FileName: "loyalsoldier-reject.json",
				URL:      "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/reject-list.txt",
			},
		},
	}
}

type sourceRuleSetFile struct {
	Version uint8        `json:"version"`
	Rules   []sourceRule `json:"rules,omitempty"`
}

type sourceRule struct {
	Domain        []string `json:"domain,omitempty"`
	DomainSuffix  []string `json:"domain_suffix,omitempty"`
	DomainKeyword []string `json:"domain_keyword,omitempty"`
	DomainRegex   []string `json:"domain_regex,omitempty"`
}

func (i *LoyalsoldierRuleSetInstaller) Install(ctx context.Context) ([]map[string]any, error) {
	if err := os.MkdirAll(i.ruleSetDir, 0700); err != nil {
		return nil, fmt.Errorf("create rule-set dir: %w", err)
	}

	entries := make([]map[string]any, 0, len(i.sources)+len(remoteRuleSetDefaults))
	for _, src := range i.sources {
		ruleFile, err := i.fetchAndConvert(ctx, src)
		if err != nil {
			return nil, err
		}

		path := filepath.Join(i.ruleSetDir, src.FileName)
		data, err := json.MarshalIndent(ruleFile, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal %s: %w", src.Tag, err)
		}
		if err := atomicWriteFile0600(path, data); err != nil {
			return nil, fmt.Errorf("write %s: %w", src.Tag, err)
		}

		entries = append(entries, map[string]any{
			"tag":    src.Tag,
			"type":   "local",
			"format": "source",
			"path":   path,
		})
	}
	entries = append(entries, remoteRuleSetDefaults...)

	return entries, nil
}

// remoteRuleSetDefaults 是以 remote 方式引用的 SagerNet 官方二进制规则集（.srs），
// 由内核自动下载并缓存，无需本地转换。这些规则集覆盖中国域名、中国 IP、广告、
// Google Play 等常见匹配场景。
var remoteRuleSetDefaults = []map[string]any{
	{
		"tag":             "geosite-cn",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs",
		"download_detour": "direct",
		"update_interval": "1d",
	},
	{
		"tag":             "geoip-cn",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs",
		"download_detour": "direct",
		"update_interval": "1d",
	},
	{
		"tag":             "geosite-google-play",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-google-play.srs",
		"download_detour": "direct",
		"update_interval": "1d",
	},
	{
		"tag":             "geosite-category-ads-all",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs",
		"download_detour": "direct",
		"update_interval": "1d",
	},
}

func DefaultRemoteRuleSetInterval() string { return "1d" }

func BuiltinLocalRuleSetTags() []string {
	return []string{"loyalsoldier-direct", "loyalsoldier-proxy", "loyalsoldier-reject"}
}

func BuiltinRemoteRuleSetTags() []string {
	return []string{"geosite-cn", "geoip-cn", "geosite-google-play", "geosite-category-ads-all"}
}

func (i *LoyalsoldierRuleSetInstaller) RuleSetDir() string { return i.ruleSetDir }

func (i *LoyalsoldierRuleSetInstaller) SourceByTag(tag string) (RuleSetSource, bool) {
	for _, src := range i.sources {
		if src.Tag == tag {
			return src, true
		}
	}
	return RuleSetSource{}, false
}

func (i *LoyalsoldierRuleSetInstaller) IsBuiltinLocal(tag string) bool {
	_, ok := i.SourceByTag(tag)
	return ok
}

func atomicWriteFile0600(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".ruleset-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func (i *LoyalsoldierRuleSetInstaller) fetchAndConvert(ctx context.Context, src RuleSetSource) (sourceRuleSetFile, error) {
	primary := src.URL
	var lastErr error
	for _, candidate := range ruleSetSourceURLs(primary) {
		ruleFile, err := i.downloadAndConvertURL(ctx, src.Tag, candidate)
		if err == nil {
			return ruleFile, nil
		}
		lastErr = err
	}
	return sourceRuleSetFile{}, lastErr
}

// ruleSetSourceURLs 返回该源的下载地址序列：主地址优先，其次为 jsDelivr
// GitHub CDN 镜像；raw.githubusercontent.com 不可达时自动退避到镜像源。
func ruleSetSourceURLs(primary string) []string {
	mirrors := jsdelivrMirrorURLs(primary)
	urls := make([]string, 0, 1+len(mirrors))
	urls = append(urls, primary)
	urls = append(urls, mirrors...)
	return urls
}

// jsdelivrMirrorURLs 将 raw.githubusercontent.com 地址映射为官方 jsDelivr
// CDN 地址（release 分支文件，见 v2ray-rules-dat README 的推荐下载地址）。
func jsdelivrMirrorURLs(rawURL string) []string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() != "raw.githubusercontent.com" {
		return nil
	}
	parts := strings.SplitN(strings.TrimPrefix(parsed.Path, "/"), "/", 4)
	if len(parts) != 4 {
		return nil
	}
	cdnURL := "https://cdn.jsdelivr.net/gh/" + parts[0] + "/" + parts[1] + "@" + parts[2] + "/" + parts[3]
	return []string{cdnURL, "https://fastly.jsdelivr.net/" + strings.TrimPrefix(cdnURL, "https://cdn.jsdelivr.net/")}
}

func (i *LoyalsoldierRuleSetInstaller) downloadAndConvertURL(ctx context.Context, tag, sourceURL string) (sourceRuleSetFile, error) {
	if err := ValidatePublicHTTPURL(sourceURL); err != nil {
		return sourceRuleSetFile{}, fmt.Errorf("download %s: %w", tag, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return sourceRuleSetFile{}, fmt.Errorf("build request %s: %w", tag, err)
	}
	client := i.client
	if client == nil {
		client = newPublicHTTPClient(ruleSetInstallerHTTPTimeout)
	}
	resp, err := client.Do(req)
	if err != nil {
		return sourceRuleSetFile{}, fmt.Errorf("download %s: %w", tag, err)
	}
	if resp == nil || resp.Body == nil {
		return sourceRuleSetFile{}, fmt.Errorf("download %s: response body is nil", tag)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return sourceRuleSetFile{}, fmt.Errorf("download %s: unexpected status %d", tag, resp.StatusCode)
	}
	if resp.ContentLength > maxRuleSetBodyBytes {
		return sourceRuleSetFile{}, fmt.Errorf("download %s: %w", tag, ErrRuleSetContentTooLarge)
	}
	content, err := readRuleSetBody(resp.Body)
	if err != nil {
		return sourceRuleSetFile{}, fmt.Errorf("read %s: %w", tag, err)
	}
	return convertRuleSetContent(tag, content)
}

func convertRuleSetContent(tag string, content []byte) (sourceRuleSetFile, error) {
	var (
		domain        []string
		domainSuffix  []string
		domainKeyword []string
		domainRegex   []string
	)

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "full:"):
			domain = append(domain, strings.TrimSpace(strings.TrimPrefix(line, "full:")))
		case strings.HasPrefix(line, "keyword:"):
			domainKeyword = append(domainKeyword, strings.TrimSpace(strings.TrimPrefix(line, "keyword:")))
		case strings.HasPrefix(line, "regexp:"):
			domainRegex = append(domainRegex, strings.TrimSpace(strings.TrimPrefix(line, "regexp:")))
		case strings.HasPrefix(line, "domain:"):
			domainSuffix = append(domainSuffix, strings.TrimSpace(strings.TrimPrefix(line, "domain:")))
		default:
			domainSuffix = append(domainSuffix, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return sourceRuleSetFile{}, fmt.Errorf("read %s: %w", tag, err)
	}

	domain = uniqueStrings(domain)
	domainSuffix = uniqueStrings(domainSuffix)
	domainKeyword = uniqueStrings(domainKeyword)
	domainRegex = uniqueStrings(domainRegex)

	rule := sourceRule{
		Domain:        domain,
		DomainSuffix:  domainSuffix,
		DomainKeyword: domainKeyword,
		DomainRegex:   domainRegex,
	}
	if len(rule.Domain) == 0 && len(rule.DomainSuffix) == 0 && len(rule.DomainKeyword) == 0 && len(rule.DomainRegex) == 0 {
		return sourceRuleSetFile{}, fmt.Errorf("source %s produced no valid rules", tag)
	}

	return sourceRuleSetFile{
		Version: C.RuleSetVersionCurrent,
		Rules:   []sourceRule{rule},
	}, nil
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}
