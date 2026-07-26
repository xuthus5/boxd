package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

const dnsOutboundBootstrapCycleConfig = `{
  "outbounds":[{
    "type":"socks","tag":"proxy","server":"proxy.example.com","server_port":1080,
    "domain_resolver":"remote"
  }],
  "dns":{"servers":[
    {"type":"udp","tag":"remote","server":"1.1.1.1","detour":"proxy"}
  ],"final":"remote"},
  "route":{"final":"proxy"}
}`

const dnsOutboundDefaultBootstrapCycleConfig = `{
  "outbounds":[{
    "type":"socks","tag":"proxy","server":"proxy.example.com","server_port":1080
  }],
  "dns":{"servers":[
    {"type":"udp","tag":"remote","server":"1.1.1.1","detour":"proxy"}
  ],"final":"remote"},
  "route":{"final":"proxy","default_domain_resolver":"remote"}
}`

const dnsOutboundDetouredBootstrapCycleConfig = `{
  "outbounds":[
    {"type":"socks","tag":"proxy","server":"proxy.example.com","server_port":1080,"detour":"underlay"},
    {"type":"direct","tag":"underlay","bind_interface":"lo"}
  ],
  "dns":{"servers":[
    {"type":"udp","tag":"remote","server":"1.1.1.1","detour":"proxy"}
  ],"final":"remote"},
  "route":{"final":"proxy","default_domain_resolver":"remote"}
}`

const missingDNSDomainResolverConfig = `{
  "dns":{"servers":[
    {"type":"udp","tag":"remote","server":"dns.example.com"}
  ]}
}`

func TestValidateConfigRejectsDNSOutboundBootstrapCycles(t *testing.T) {
	for _, body := range []string{
		dnsOutboundBootstrapCycleConfig,
		dnsOutboundDefaultBootstrapCycleConfig,
		dnsOutboundDetouredBootstrapCycleConfig,
	} {
		handler, _ := newConfigHandlerWithFile(t)
		recorder := httptest.NewRecorder()
		handler.ValidateConfig(
			recorder,
			jsonRequest(http.MethodPost, "/api/config/validate", body),
		)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
		}
		envelope := decodeEnvelope(t, recorder)
		if envelope.Error == nil || envelope.Error.Code != model.ErrorConfigInvalidRuntime {
			t.Fatalf("error = %#v", envelope.Error)
		}
		want := `dns.servers[0].detour: DNS dependency cycle through "proxy"`
		if envelope.Error.Message != want {
			t.Fatalf("message = %q, want %q", envelope.Error.Message, want)
		}
	}
}

func TestValidateConfigRejectsMissingDNSDomainResolver(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)
	recorder := httptest.NewRecorder()
	handler.ValidateConfig(
		recorder,
		jsonRequest(http.MethodPost, "/api/config/validate", missingDNSDomainResolverConfig),
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	envelope := decodeEnvelope(t, recorder)
	if envelope.Error == nil || envelope.Error.Code != model.ErrorConfigInvalidRuntime {
		t.Fatalf("error = %#v", envelope.Error)
	}
	want := `dns.servers[0].server: DNS server "remote" requires a domain resolver or detour`
	if envelope.Error.Message != want {
		t.Fatalf("message = %q, want %q", envelope.Error.Message, want)
	}
}

func TestConfigSaveRejectsDNSOutboundBootstrapCyclesBeforeSideEffects(t *testing.T) {
	assertConfigSaveRejectedBeforeSideEffects(t, dnsOutboundDetouredBootstrapCycleConfig)
}

func TestConfigSaveRejectsMissingDNSDomainResolverBeforeSideEffects(t *testing.T) {
	assertConfigSaveRejectedBeforeSideEffects(t, missingDNSDomainResolverConfig)
}

func assertConfigSaveRejectedBeforeSideEffects(t *testing.T, body string) {
	t.Helper()
	tests := []struct {
		name   string
		method string
		path   string
		handle func(*ConfigHandler, http.ResponseWriter, *http.Request)
	}{
		{name: "structured save", method: http.MethodPut, path: "/api/config", handle: (*ConfigHandler).UpdateConfig},
		{name: "raw save", method: http.MethodPut, path: "/api/config/raw", handle: (*ConfigHandler).UpdateRawConfig},
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
			test.handle(
				handler,
				recorder,
				jsonRequest(test.method, test.path, body),
			)

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
