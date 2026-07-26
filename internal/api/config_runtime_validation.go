package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

var ErrInvalidRuntimeConfig = errors.New("invalid sing-box config")

type runtimeDiagnosticTemplate struct {
	message      string
	valueMessage string
}

var runtimeDiagnosticTemplates = map[string]runtimeDiagnosticTemplate{
	"duplicate_tag": {message: "duplicate tag", valueMessage: "duplicate tag %q"},
	"missing_tag":   {message: "tag is required"},
	"empty_group": {
		message:      "outbound group must contain at least one member",
		valueMessage: "%q outbound group must contain at least one member",
	},
	"invalid_group_default": {
		message:      "selector default is not a group member",
		valueMessage: "selector default %q is not a group member",
	},
	"outbound_dependency_cycle": {
		message:      "outbound dependency cycle",
		valueMessage: "outbound dependency cycle through %q",
	},
	"dns_dependency_cycle": {message: "DNS dependency cycle", valueMessage: "DNS dependency cycle through %q"},
	"invalid_dns_default": {
		message:      "FakeIP DNS server cannot be the default",
		valueMessage: "FakeIP DNS server %q cannot be the default",
	},
	"multiple_fakeip_dns_servers": {
		message:      "multiple FakeIP DNS servers are not supported",
		valueMessage: "multiple FakeIP DNS servers are not supported; remove %q",
	},
	"unknown_outbound_reference": {message: "unknown outbound reference", valueMessage: "unknown outbound reference %q"},
	"unknown_ruleset_reference":  {message: "unknown rule-set reference", valueMessage: "unknown rule-set reference %q"},
	"unknown_dns_reference": {
		message:      "unknown DNS server reference",
		valueMessage: "unknown DNS server reference %q",
	},
}

func validateRuntimeConfig(body []byte) error {
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	defer cancel()

	var cfg option.Options
	if err := cfg.UnmarshalJSONContext(ctx, body); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRuntimeConfig, err)
	}
	for _, issue := range core.AnalyzeConfig(body).Issues {
		if issue.Severity == model.ConfigDiagnosticSeverityError {
			return fmt.Errorf("%w: %s", ErrInvalidRuntimeConfig, runtimeDiagnosticMessage(issue))
		}
	}
	return nil
}

func runtimeDiagnosticMessage(issue model.ConfigDiagnostic) string {
	detail := strings.TrimSpace(issue.Detail)
	if detail == "" {
		detail = runtimeDiagnosticDetail(issue)
	}
	path := strings.TrimSpace(issue.Path)
	if path == "" {
		return detail
	}
	return path + ": " + detail
}

func runtimeDiagnosticDetail(issue model.ConfigDiagnostic) string {
	template, exists := runtimeDiagnosticTemplates[issue.Code]
	if !exists {
		return fallbackRuntimeDiagnosticDetail(issue.Code)
	}
	value := strings.TrimSpace(issue.Value)
	if value != "" && template.valueMessage != "" {
		return fmt.Sprintf(template.valueMessage, value)
	}
	return template.message
}

func fallbackRuntimeDiagnosticDetail(code string) string {
	detail := strings.TrimSpace(strings.ReplaceAll(code, "_", " "))
	if detail == "" {
		return "invalid sing-box config"
	}
	return detail
}

func runtimeConfigErrorMessage(err error) string {
	msg := strings.TrimSpace(err.Error())
	prefix := ErrInvalidRuntimeConfig.Error() + ": "
	if detail, ok := strings.CutPrefix(msg, prefix); ok {
		msg = strings.TrimSpace(detail)
	}
	if lines := strings.Split(msg, "\n"); len(lines) > 1 {
		msg = singleLineRuntimeConfigError(lines)
	}
	if msg == "" {
		return "invalid sing-box config"
	}
	return strings.TrimSpace(msg)
}

func singleLineRuntimeConfigError(lines []string) string {
	first := strings.TrimSpace(lines[0])
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, ".") || strings.Contains(line, "[") {
			return first + ": " + line
		}
	}
	return first
}
