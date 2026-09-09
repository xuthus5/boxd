package service

import (
	"os"
	"testing"
)

func TestConfigStoresSingBox114SharedResources(t *testing.T) {
	svc := newTestService(t)
	body := []byte(`{"dns":{"servers":[{"type":"local","tag":"resolver"}]},
"route":{"default_domain_resolver":"resolver","default_http_client":"downloads"},
"http_clients":[{"tag":"downloads","version":2}],
"certificate_providers":[{"type":"cloudflare-origin-ca","tag":"origin","domain":["example.invalid"],"api_token":"test-token","http_client":"downloads"}],
"network_namespaces":[{"type":"default","tag":"isolated","path":"/proc/self/ns/net"}]}`)
	result, err := svc.Config().ApplyConfig(t.Context(), body, "shared_resources")
	if err != nil || result.Status != "ok" {
		t.Fatalf("save shared resources: %+v %v", result, err)
	}
	saved, err := os.ReadFile(svc.Deps.ConfigPath)
	if err != nil || string(saved) != string(body) {
		t.Fatalf("shared resource round-trip failed: %v", err)
	}
}
