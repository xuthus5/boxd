package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSetupStatusReportsMissingAndInvalidConfigurations(t *testing.T) {
	env := setupTestEnvironment(t)
	for _, tt := range []struct {
		name    string
		body    string
		invalid bool
	}{
		{name: "missing"},
		{name: "empty object", body: `{}`},
		{name: "invalid JSON", body: `{`, invalid: true},
		{name: "non-object", body: `[]`, invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			status := InspectSetup([]byte(tt.body), env)
			if (status.ConfigError != "") != tt.invalid {
				t.Fatalf("config error = %q", status.ConfigError)
			}
			if len(status.Modules) != 6 || status.Listeners == nil || status.ProxyReady {
				t.Fatalf("incomplete setup status: %#v", status)
			}
			for _, module := range status.Modules {
				if module.Dependencies == nil {
					t.Fatal("dependencies must serialize as arrays")
				}
			}
		})
	}
}

func TestSetupStatusDetectsCapabilityAndReferenceErrors(t *testing.T) {
	env := setupTestEnvironment(t)
	env.Capabilities.TUNReason = NetworkTUNDevice
	body := []byte(`{"inbounds":[{"type":"tun","tag":"tun-in","address":["172.19.0.1/30"]}],"outbounds":[{"type":"selector","tag":"proxy","outbounds":["missing"]}],"route":{"final":"proxy"}}`)
	status := InspectSetup(body, env)
	states := map[string]string{}
	for _, module := range status.Modules {
		states[module.ID] = module.State
	}
	if states["inbounds"] != "invalid" || states["outbounds"] != "invalid" {
		t.Fatalf("invalid modules not reported: %#v", status.Modules)
	}
	if status.ProxyReady {
		t.Fatal("a broken selector is not proxy-ready")
	}
}

func TestSetupStatusListsClientAccessWithoutCredentials(t *testing.T) {
	body := []byte(`{"inbounds":[{"type":"mixed","tag":"local","listen":"127.0.0.1","listen_port":2080,"users":[{"username":"private","password":"secret"}]}],"outbounds":[{"type":"socks","tag":"node","server":"1.2.3.4","server_port":1080}]}`)
	status := InspectSetup(body, setupTestEnvironment(t))
	if !status.ProxyReady || len(status.Listeners) != 1 || status.Listeners[0].ListenPort != 2080 {
		t.Fatalf("incorrect client access details: %#v", status)
	}
	encoded, err := json.Marshal(status.Listeners)
	if err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal(encoded, &items); err != nil {
		t.Fatal(err)
	}
	if _, exists := items[0]["users"]; exists {
		t.Fatal("setup status exposed inbound credentials")
	}
}

func TestSetupStatusExposesSemanticErrorsForExplicitRebuild(t *testing.T) {
	body := []byte(`{"outbounds":[{"type":"direct","tag":"direct"}],"route":{"final":"missing"}}`)
	status := InspectSetup(body, setupTestEnvironment(t))
	if status.ConfigError == "" {
		t.Fatal("semantic errors must expose the explicit rebuild option")
	}
	for _, module := range status.Modules {
		if module.ID == "route" && module.State != "invalid" {
			t.Fatal("the invalid module must remain identifiable")
		}
		if module.ID == "experimental" && module.State != "missing" {
			t.Fatal("a route error must not invalidate unrelated modules")
		}
	}
}

func TestSetupStatusChecksConfiguredTUNAddressFamilies(t *testing.T) {
	for _, tt := range []struct {
		name    string
		address string
		reason  string
		invalid bool
	}{
		{name: "disabled dual stack", address: "fdfe:dcba:9876::1/126", reason: NetworkIPv6Disabled, invalid: true},
		{name: "unsupported dual stack", address: "fdfe:dcba:9876::1/126", reason: NetworkIPv6Unsupported, invalid: true},
		{name: "unknown dual stack", address: "fdfe:dcba:9876::1/126", reason: NetworkIPv6Unknown},
		{name: "disabled IPv4 only", address: "172.19.0.1/30", reason: NetworkIPv6Disabled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTestEnvironment(t)
			env.Capabilities.TUNAvailable = true
			env.Capabilities.IPv6Available = false
			env.Capabilities.IPv6Reason = tt.reason
			body := []byte(`{"inbounds":[{"type":"tun","tag":"tun-in","address":["` + tt.address + `"]}]}`)
			status := InspectSetup(body, env)
			for _, module := range status.Modules {
				if module.ID == "inbounds" && (module.State == "invalid") != tt.invalid {
					t.Fatalf("configured TUN capability status = %#v", module)
				}
			}
		})
	}
}

func TestReadSetupSourceHandlesFilesAndErrors(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.json")
	if body, err := ReadSetupSource(missing); err != nil || body != nil {
		t.Fatalf("missing config: %q, %v", body, err)
	}
	if _, err := ReadSetupSource(dir); err == nil {
		t.Fatal("directory read errors must be reported")
	}
	if err := os.WriteFile(missing, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if body, err := ReadSetupSource(missing); err != nil || string(body) != `{}` {
		t.Fatalf("read config: %q, %v", body, err)
	}
}
