package core

import (
	"net"
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/adapter"
)

func ruleName(rule adapter.Rule) string {
	if rule == nil {
		return ""
	}
	typeName := ""
	if typed, ok := any(rule).(interface{ Type() string }); ok {
		typeName = typed.Type()
	}
	return formatRuleName(typeName, rule.String())
}

// formatRuleName prefers a human-readable rule expression over generic types.
func formatRuleName(typeName, raw string) string {
	expression := compactRuleExpression(raw)
	kind := strings.TrimSpace(typeName)
	if expression == "" {
		return kind
	}
	// "default"/"logical" alone are not useful in connection lists.
	if kind == "" || kind == "default" || kind == "logical" {
		return truncateRuleName(expression, 120)
	}
	if strings.EqualFold(expression, kind) {
		return kind
	}
	return truncateRuleName(kind+": "+expression, 120)
}

func compactRuleExpression(raw string) string {
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}

func truncateRuleName(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if limit <= 0 || len(runes) <= limit {
		return string(runes)
	}
	if limit <= 1 {
		return string(runes[:limit])
	}
	return string(runes[:limit-1]) + "…"
}

func connectionTarget(metadata adapter.InboundContext) string {
	if domain := strings.TrimSpace(metadata.Domain); domain != "" {
		return formatHostPort(domain, metadata.Destination.Port)
	}
	if host := strings.TrimSpace(metadata.Destination.Fqdn); host != "" {
		return formatHostPort(host, metadata.Destination.Port)
	}
	if metadata.Destination.Addr.IsValid() {
		return formatHostPort(metadata.Destination.Addr.String(), metadata.Destination.Port)
	}
	return ""
}

func formatHostPort(host string, port uint16) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if port == 0 {
		return host
	}
	return net.JoinHostPort(host, strconv.Itoa(int(port)))
}

func connectionSource(metadata adapter.InboundContext) string {
	if metadata.Source.IsValid() {
		return metadata.Source.String()
	}
	return ""
}

func connectionNetwork(metadata adapter.InboundContext) string {
	return strings.TrimSpace(metadata.Network)
}

func connectionInbound(metadata adapter.InboundContext) string {
	if tag := strings.TrimSpace(metadata.Inbound); tag != "" {
		return tag
	}
	return strings.TrimSpace(metadata.InboundType)
}

func connectionProtocol(metadata adapter.InboundContext) string {
	return strings.TrimSpace(metadata.Protocol)
}

func connectionProcess(metadata adapter.InboundContext) string {
	info := metadata.ProcessInfo
	if info == nil {
		return ""
	}
	if path := strings.TrimSpace(info.ProcessPath); path != "" {
		return path
	}
	if name := strings.TrimSpace(info.UserName); name != "" {
		return name
	}
	if len(info.AndroidPackageNames) > 0 {
		return strings.TrimSpace(info.AndroidPackageNames[0])
	}
	return ""
}
