package core

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

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
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
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
		if err := os.WriteFile(path, data, 0600); err != nil {
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
	},
	{
		"tag":             "geoip-cn",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs",
		"download_detour": "direct",
	},
	{
		"tag":             "geosite-google-play",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-google-play.srs",
		"download_detour": "direct",
	},
	{
		"tag":             "geosite-category-ads-all",
		"type":            "remote",
		"format":          "binary",
		"url":             "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs",
		"download_detour": "direct",
	},
}

func (i *LoyalsoldierRuleSetInstaller) fetchAndConvert(ctx context.Context, src RuleSetSource) (sourceRuleSetFile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
	if err != nil {
		return sourceRuleSetFile{}, fmt.Errorf("build request %s: %w", src.Tag, err)
	}
	resp, err := i.client.Do(req)
	if err != nil {
		return sourceRuleSetFile{}, fmt.Errorf("download %s: %w", src.Tag, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return sourceRuleSetFile{}, fmt.Errorf("download %s: unexpected status %d", src.Tag, resp.StatusCode)
	}

	var (
		domain        []string
		domainSuffix  []string
		domainKeyword []string
		domainRegex   []string
	)

	scanner := bufio.NewScanner(resp.Body)
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
		return sourceRuleSetFile{}, fmt.Errorf("read %s: %w", src.Tag, err)
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
		return sourceRuleSetFile{}, fmt.Errorf("source %s produced no valid rules", src.Tag)
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
