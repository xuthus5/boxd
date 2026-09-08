package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	C "github.com/sagernet/sing-box/constant"
)

type RuleSetDefaultsInstaller interface {
	Install(ctx context.Context) ([]map[string]any, error)
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
		sources:    defaultRuleSetSources(),
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(i.ruleSetDir, 0700); err != nil {
		return nil, fmt.Errorf("create rule-set dir: %w", err)
	}

	entries := make([]map[string]any, 0, len(i.sources))
	for _, src := range i.sources {
		data, err := i.downloadRuleSet(ctx, src)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := atomicWriteFile0600(filepath.Join(i.ruleSetDir, src.FileName), data); err != nil {
			return nil, fmt.Errorf("write %s: %w", src.Tag, err)
		}
		entries = append(entries, localRuleSetEntry(i.ruleSetDir, src))
	}

	return entries, nil
}

func localRuleSetEntry(dir string, src RuleSetSource) map[string]any {
	format := "source"
	if src.Format == "binary" || src.Format == "cidr" {
		format = "binary"
	}
	return map[string]any{
		"tag": src.Tag, "type": "local", "format": format,
		"path": filepath.Join(dir, src.FileName),
	}
}

func (i *LoyalsoldierRuleSetInstaller) downloadRuleSet(ctx context.Context, src RuleSetSource) ([]byte, error) {
	content, err := i.raceDownloadContent(ctx, src.Tag, src.URL)
	if err != nil {
		return nil, err
	}
	if src.Format == "cidr" {
		ipv6, err := i.raceDownloadContent(ctx, src.Tag, strings.TrimSuffix(src.URL, ".txt")+"6.txt")
		if err != nil {
			return nil, err
		}
		content = append(append(content, '\n'), ipv6...)
	}
	return convertRuleSetData(src, content)
}

func convertRuleSetData(src RuleSetSource, content []byte) ([]byte, error) {
	switch src.Format {
	case "binary":
		return content, nil
	case "cidr":
		return convertCIDRRuleSet(src.Tag, content)
	default:
		ruleFile, err := convertRuleSetContent(src.Tag, content)
		if err != nil {
			return nil, err
		}
		data, err := json.MarshalIndent(ruleFile, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal %s: %w", src.Tag, err)
		}
		return data, nil
	}
}

func (i *LoyalsoldierRuleSetInstaller) fetchAndConvert(ctx context.Context, src RuleSetSource) (sourceRuleSetFile, error) {
	content, err := i.raceDownloadContent(ctx, src.Tag, src.URL)
	if err != nil {
		return sourceRuleSetFile{}, err
	}
	return convertRuleSetContent(src.Tag, content)
}

func convertRuleSetContent(tag string, content []byte) (sourceRuleSetFile, error) {
	var rule sourceRule
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		appendRuleSetLine(&rule, line)
	}
	if err := scanner.Err(); err != nil {
		return sourceRuleSetFile{}, fmt.Errorf("read %s: %w", tag, err)
	}

	rule.Domain = uniqueStrings(rule.Domain)
	rule.DomainSuffix = uniqueStrings(rule.DomainSuffix)
	rule.DomainKeyword = uniqueStrings(rule.DomainKeyword)
	rule.DomainRegex = uniqueStrings(rule.DomainRegex)
	if len(rule.Domain)+len(rule.DomainSuffix)+len(rule.DomainKeyword)+len(rule.DomainRegex) == 0 {
		return sourceRuleSetFile{}, fmt.Errorf("source %s produced no valid rules", tag)
	}

	return sourceRuleSetFile{
		Version: C.RuleSetVersionCurrent,
		Rules:   []sourceRule{rule},
	}, nil
}

func appendRuleSetLine(rule *sourceRule, line string) {
	switch {
	case strings.HasPrefix(line, "full:"):
		rule.Domain = append(rule.Domain, strings.TrimSpace(strings.TrimPrefix(line, "full:")))
	case strings.HasPrefix(line, "keyword:"):
		rule.DomainKeyword = append(rule.DomainKeyword, strings.TrimSpace(strings.TrimPrefix(line, "keyword:")))
	case strings.HasPrefix(line, "regexp:"):
		rule.DomainRegex = append(rule.DomainRegex, strings.TrimSpace(strings.TrimPrefix(line, "regexp:")))
	case strings.HasPrefix(line, "domain:"):
		rule.DomainSuffix = append(rule.DomainSuffix, strings.TrimSpace(strings.TrimPrefix(line, "domain:")))
	default:
		rule.DomainSuffix = append(rule.DomainSuffix, line)
	}
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
