package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/route/rule"

	"github.com/xuthus5/boxd/internal/model"
)

func addDiagnosticFileRuleSetShapes(shapes map[string]diagnosticRuleSetShape, entry diagnosticObject) {
	if stringValue(entry.object["type"]) == "remote" {
		// 远程种子可能已被缓存或下载结果替代，不能据此断言当前内容。
		return
	}
	path := stringValue(entry.object["path"])
	if path == "" {
		return
	}
	for _, tag := range stringValues(entry.object["tag"]) {
		content, err := readDiagnosticRuleSetFile(strings.ReplaceAll(path, "{tag}", tag))
		if err != nil {
			continue
		}
		parsed, err := parseDiagnosticRuleSet(content, diagnosticRuleSetFormat(entry, path))
		if err == nil {
			shapes[tag] = diagnosticShapeForRules(parsed.Rules)
		}
	}
}

func checkPersistedRuleSetFiles(report *model.ConfigDiagnostics, body []byte) {
	var cfg map[string]any
	if err := json.Unmarshal(body, &cfg); err != nil {
		return
	}
	for _, entry := range diagnosticObjects(objectValue(cfg["route"])["rule_set"], "route.rule_set") {
		if stringValue(entry.object["type"]) != "local" {
			continue
		}
		for _, tag := range stringValues(entry.object["tag"]) {
			checkPersistedRuleSetFile(report, entry, tag)
		}
	}
}

func checkPersistedRuleSetFile(report *model.ConfigDiagnostics, entry diagnosticObject, tag string) {
	path := strings.ReplaceAll(stringValue(entry.object["path"]), "{tag}", tag)
	content, err := readDiagnosticRuleSetFile(path)
	if err != nil {
		code := "ruleset_file_unreadable"
		if errors.Is(err, os.ErrNotExist) {
			code = "ruleset_file_missing"
		}
		addDiagnostic(report, code, model.ConfigDiagnosticSeverityError, entry.path+".path", tag, "")
		return
	}
	if _, err := parseDiagnosticRuleSet(content, diagnosticRuleSetFormat(entry, path)); err != nil {
		addDiagnostic(report, "ruleset_file_invalid", model.ConfigDiagnosticSeverityError, entry.path+".path", tag, "")
	}
}

func readDiagnosticRuleSetFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("rule-set path is not a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return readRuleSetBody(file)
}

func diagnosticRuleSetFormat(entry diagnosticObject, path string) string {
	if format := stringValue(entry.object["format"]); format != "" {
		return format
	}
	if stringValue(entry.object["type"]) == "remote" {
		path = stringValue(entry.object["url"])
	}
	if parsed, err := url.Parse(path); err == nil {
		path = parsed.Path
	}
	if filepath.Ext(path) == ".srs" {
		return "binary"
	}
	return "source"
}

func parseDiagnosticRuleSet(content []byte, format string) (option.PlainRuleSet, error) {
	var parsed option.PlainRuleSetCompat
	var err error
	if format == "binary" {
		parsed, err = srs.Read(bytes.NewReader(content), false)
	} else {
		err = json.Unmarshal(content, &parsed)
	}
	if err != nil {
		return option.PlainRuleSet{}, err
	}
	return parsed.Upgrade()
}

func diagnosticShapeForRules(rules []option.HeadlessRule) diagnosticRuleSetShape {
	return diagnosticRuleSetShape{
		ip: rule.HasHeadlessRule(rules, func(item option.DefaultHeadlessRule) bool {
			return len(item.IPCIDR) > 0 || item.IPSet != nil
		}),
		query: rule.HasHeadlessRule(rules, func(item option.DefaultHeadlessRule) bool { return len(item.QueryType) > 0 }),
		other: rule.HasHeadlessRule(rules, func(item option.DefaultHeadlessRule) bool {
			ipOnly := option.DefaultHeadlessRule{IPCIDR: item.IPCIDR, IPSet: item.IPSet, Invert: item.Invert}
			return !reflect.DeepEqual(item, ipOnly)
		}),
	}
}
