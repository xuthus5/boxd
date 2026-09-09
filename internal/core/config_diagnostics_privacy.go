package core

import (
	"errors"
	"slices"
	"strconv"
	"strings"
)

func diagnosticConfigErrorDetail(err error, cfg map[string]any) string {
	if err == nil {
		return ""
	}
	var secrets []string
	collectDiagnosticSecrets(cfg, false, &secrets)
	slices.SortFunc(secrets, func(a, b string) int { return len(b) - len(a) })
	detail := err.Error()
	for _, secret := range secrets {
		quoted := strconv.Quote(secret)
		detail = strings.ReplaceAll(detail, quoted[1:len(quoted)-1], "[redacted]")
		detail = strings.ReplaceAll(detail, secret, "[redacted]")
	}
	return diagnosticDetail(errors.New(detail))
}

func collectDiagnosticSecrets(value any, sensitive bool, secrets *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			collectDiagnosticSecrets(child, sensitive || sensitiveDiagnosticKey(key), secrets)
		}
	case []any:
		for _, child := range typed {
			collectDiagnosticSecrets(child, sensitive, secrets)
		}
	case string:
		if sensitive && typed != "" {
			*secrets = append(*secrets, typed)
		}
	}
}

func sensitiveDiagnosticKey(key string) bool {
	key = strings.ToLower(key)
	switch key {
	case "uuid", "password", "secret", "token", "cookie", "psk", "key", "authorization", "headers", "credentials":
		return true
	default:
		return strings.HasSuffix(key, "_key") || strings.HasSuffix(key, "_secret") || strings.HasSuffix(key, "_token") || strings.HasSuffix(key, "_password")
	}
}
