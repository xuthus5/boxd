package api

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

const outboundDependencyCycleConfig = `{
  "outbounds":[
    {"type":"selector","tag":"group-a","outbounds":["group-b"]},
    {"type":"urltest","tag":"group-b","outbounds":["group-a"]}
  ]
}`

const dnsDependencyCycleConfig = `{
  "dns":{"servers":[
    {"type":"https","tag":"modern-a","server":"1.1.1.1","domain_resolver":"modern-b"},
    {"type":"https","tag":"modern-b","server":"1.0.0.1","domain_resolver":{"server":"modern-a"}}
  ],"final":"modern-a"}
}`

func TestValidateConfigRejectsSemanticRuntimeErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		message string
	}{
		{
			name:    "outbound dependency cycle",
			body:    outboundDependencyCycleConfig,
			message: `outbounds[1].outbounds[0]: outbound dependency cycle through "group-a"`,
		},
		{
			name:    "DNS dependency cycle",
			body:    dnsDependencyCycleConfig,
			message: `dns.servers[1].domain_resolver.server: DNS dependency cycle through "modern-a"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, _ := newConfigHandlerWithFile(t)
			recorder := httptest.NewRecorder()
			handler.ValidateConfig(
				recorder,
				jsonRequest(http.MethodPost, "/api/config/validate", test.body),
			)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
			}
			envelope := decodeEnvelope(t, recorder)
			if envelope.Error == nil || envelope.Error.Code != model.ErrorConfigInvalidRuntime {
				t.Fatalf("error = %#v", envelope.Error)
			}
			if envelope.Error.Message != test.message {
				t.Fatalf("message = %q, want %q", envelope.Error.Message, test.message)
			}
		})
	}
}

func TestConfigSaveRejectsSemanticRuntimeErrorsBeforeSideEffects(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		handle func(*ConfigHandler, http.ResponseWriter, *http.Request)
	}{
		{
			name:   "structured save",
			method: http.MethodPut,
			path:   "/api/config",
			body:   outboundDependencyCycleConfig,
			handle: (*ConfigHandler).UpdateConfig,
		},
		{
			name:   "raw save",
			method: http.MethodPut,
			path:   "/api/config/raw",
			body:   dnsDependencyCycleConfig,
			handle: (*ConfigHandler).UpdateRawConfig,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			initial := []byte(`{"log":{"level":"info"}}`)
			configPath := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(configPath, initial, 0600); err != nil {
				t.Fatal(err)
			}
			instance := &fakeRestartable{}
			handler := NewConfigHandler(configPath, instance, nil, nil, nil, nil)
			recorder := httptest.NewRecorder()
			test.handle(handler, recorder, jsonRequest(test.method, test.path, test.body))

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
			}
			assertConfigUnchanged(t, configPath, initial)
			if instance.calls != 0 {
				t.Fatalf("restart calls = %d, want 0", instance.calls)
			}
		})
	}
}

func TestRestoreConfigRejectsSemanticRuntimeErrorsBeforeSideEffects(t *testing.T) {
	handler, history, configPath := setupApplyHistoryHandler(t)
	initial, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(outboundDependencyCycleConfig)
	event := core.NewConfigApplyEvent("update", model.StatusOK, body, nil)
	if err := history.AppendSnapshot(event, body); err != nil {
		t.Fatal(err)
	}
	instance := &fakeRestartable{}
	handler.instance = instance

	recorder := httptest.NewRecorder()
	request := withURLParam(
		httptest.NewRequest(http.MethodPost, "/api/config/apply-history/restore/restore", nil),
		"id",
		event.ID,
	)
	handler.RestoreConfig(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	assertConfigUnchanged(t, configPath, initial)
	if instance.calls != 0 {
		t.Fatalf("restart calls = %d, want 0", instance.calls)
	}
}

func TestValidateRuntimeConfigAllowsDiagnosticWarnings(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "no inbounds", body: `{"outbounds":[{"type":"direct","tag":"direct"}]}`},
		{name: "no outbounds", body: `{"inbounds":[{"type":"mixed","tag":"mixed","listen":"127.0.0.1","listen_port":1080}]}`},
		{
			name: "migration warning",
			body: `{
  "outbounds":[{
    "type":"socks","tag":"proxy","server":"192.0.2.1","server_port":1080,
    "domain_strategy":"prefer_ipv4"
  }]
}`,
		},
		{
			name: "insecure TLS warning",
			body: `{
  "outbounds":[{
    "type":"trojan","tag":"proxy","server":"1.1.1.1","server_port":443,"password":"secret",
    "tls":{"enabled":true,"server_name":"example.com","insecure":true}
  }]
}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateRuntimeConfig([]byte(test.body)); err != nil {
				t.Fatalf("validateRuntimeConfig() error = %v", err)
			}
		})
	}
}

func TestRuntimeDiagnosticMessageTemplates(t *testing.T) {
	tests := []struct {
		code  string
		value string
		want  string
	}{
		{code: "duplicate_tag", value: "same", want: `config.path: duplicate tag "same"`},
		{code: "missing_tag", want: "config.path: tag is required"},
		{
			code: "empty_group", value: "selector",
			want: `config.path: "selector" outbound group must contain at least one member`,
		},
		{
			code: "invalid_group_default", value: "direct",
			want: `config.path: selector default "direct" is not a group member`,
		},
		{code: "outbound_dependency_cycle", value: "group", want: `config.path: outbound dependency cycle through "group"`},
		{code: "dns_dependency_cycle", value: "dns-a", want: `config.path: DNS dependency cycle through "dns-a"`},
		{code: "invalid_dns_default", value: "fake", want: `config.path: FakeIP DNS server "fake" cannot be the default`},
		{
			code:  "multiple_fakeip_dns_servers",
			value: "fake-b",
			want:  `config.path: multiple FakeIP DNS servers are not supported; remove "fake-b"`,
		},
		{code: "unknown_outbound_reference", value: "proxy", want: `config.path: unknown outbound reference "proxy"`},
		{code: "unknown_ruleset_reference", value: "geo", want: `config.path: unknown rule-set reference "geo"`},
		{code: "unknown_dns_reference", value: "remote", want: `config.path: unknown DNS server reference "remote"`},
	}

	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			issue := model.ConfigDiagnostic{Code: test.code, Path: "config.path", Value: test.value}
			if got := runtimeDiagnosticMessage(issue); got != test.want {
				t.Fatalf("runtimeDiagnosticMessage() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRuntimeDiagnosticMessageFallbacks(t *testing.T) {
	tests := []struct {
		name  string
		issue model.ConfigDiagnostic
		want  string
	}{
		{
			name:  "detail wins",
			issue: model.ConfigDiagnostic{Code: "unknown_code", Path: "config", Detail: "runtime rejected value"},
			want:  "config: runtime rejected value",
		},
		{
			name:  "unknown code",
			issue: model.ConfigDiagnostic{Code: "future_runtime_error"},
			want:  "future runtime error",
		},
		{name: "empty issue", issue: model.ConfigDiagnostic{}, want: "invalid sing-box config"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := runtimeDiagnosticMessage(test.issue); got != test.want {
				t.Fatalf("runtimeDiagnosticMessage() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRuntimeConfigErrorMessageFallbacks(t *testing.T) {
	if got := runtimeConfigErrorMessage(errors.New("")); got != "invalid sing-box config" {
		t.Fatalf("empty error message = %q", got)
	}
	lines := []string{"decode config failed", " ", "invalid field"}
	if got := singleLineRuntimeConfigError(lines); got != "decode config failed" {
		t.Fatalf("singleLineRuntimeConfigError() = %q", got)
	}
}

func assertConfigUnchanged(t *testing.T, path string, expected []byte) {
	t.Helper()
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatalf("config = %s, want %s", actual, expected)
	}
}
