package core

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestAnalyzeConfigRedactsMalformedCredentialValues(t *testing.T) {
	const secret = "private-validation-value"
	report := AnalyzeConfig([]byte(`{
  "outbounds":[{"type":"socks","tag":"node","server":"192.0.2.1","password":"` + secret + `","domain_strategy":"` + secret + `"}]
}`))
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), secret) {
		t.Fatalf("diagnostic leaked configured credential: %s", encoded)
	}
	if !hasDiagnostic(report.Issues, "invalid_singbox_config", "") {
		t.Fatalf("invalid configuration was not diagnosed: %+v", report.Issues)
	}
}

func TestDiagnosticConfigErrorDetailRedactsNestedSecretsBeforeTruncating(t *testing.T) {
	secret := strings.Repeat("private-key", 30)
	cfg := map[string]any{
		"certificate_providers": []any{map[string]any{"api_token": secret}},
		"outbounds":             []any{map[string]any{"tls": map[string]any{"key": []any{"client-key"}}}},
		"http_clients":          []any{map[string]any{"headers": map[string]any{"X-Api": []any{"header-secret"}}}},
	}
	detail := diagnosticConfigErrorDetail(errors.New(secret+" client-key header-secret"), cfg)
	if detail != "[redacted] [redacted] [redacted]" {
		t.Fatalf("redacted detail = %q", detail)
	}
	if got := diagnosticConfigErrorDetail(nil, cfg); got != "" {
		t.Fatalf("nil error detail = %q", got)
	}
}

func TestSensitiveDiagnosticKeys(t *testing.T) {
	for _, key := range []string{"password", "UUID", "client_secret", "password_key", "auth_token", "api_password", "headers"} {
		if !sensitiveDiagnosticKey(key) {
			t.Fatalf("credential key not recognized: %s", key)
		}
	}
	for _, key := range []string{"type", "tag", "server", "route"} {
		if sensitiveDiagnosticKey(key) {
			t.Fatalf("structural key misclassified: %s", key)
		}
	}
}
