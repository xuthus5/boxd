package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

const sharedResourcesConfig = `{
  "dns":{"servers":[{"type":"local","tag":"resolver"}],"final":"resolver"},
  "outbounds":[{"type":"direct","tag":"direct"}],
  "route":{"final":"direct","default_domain_resolver":"resolver","default_http_client":"downloads"},
  "http_clients":[{"tag":"downloads","version":2}],
  "certificate_providers":[{"type":"cloudflare-origin-ca","tag":"origin","domain":["example.invalid"],"api_token":"test-token","http_client":"downloads"}],
  "network_namespaces":[{"type":"default","tag":"isolated","path":"/proc/self/ns/net"}]
}`

func TestConfigRoundTripsSingBox114SharedResources(t *testing.T) {
	handler, path := setupHandler(t, `{}`)
	kernel := &setupKernel{}
	handler.instance = kernel
	response := httptest.NewRecorder()
	handler.UpdateConfig(response, jsonRequest(http.MethodPut, "/", sharedResourcesConfig))
	if response.Code != http.StatusOK || kernel.running || kernel.reloads != 0 {
		t.Fatalf("save shared resources: %d %s", response.Code, response.Body.String())
	}
	var expected map[string]any
	if err := json.Unmarshal([]byte(sharedResourcesConfig), &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, decodeConfigFile(t, path)) {
		t.Fatal("saving lost or changed a new shared resource")
	}
}

func TestConfigRejectsMissingHTTPClientBeforeWriting(t *testing.T) {
	handler, path := setupHandler(t, `{}`)
	invalid := strings.ReplaceAll(sharedResourcesConfig, `"http_client":"downloads"`, `"http_client":"missing"`)
	response := httptest.NewRecorder()
	handler.UpdateConfig(response, jsonRequest(http.MethodPut, "/", invalid))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "http_client") {
		t.Fatalf("expected actionable shared reference error: %d %s", response.Code, response.Body.String())
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != `{}` {
		t.Fatalf("invalid shared resources changed config: %v", err)
	}
}
