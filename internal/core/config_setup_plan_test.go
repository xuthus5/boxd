package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func setupTestEnvironment(t *testing.T) SetupEnvironment {
	t.Helper()
	return SetupEnvironment{DataDir: filepath.Join(t.TempDir(), "data"), Capabilities: NetworkCapabilities{
		Platform: "linux", DefaultListen: "127.0.0.1", IPv6Available: true,
	}}
}

func TestSetupPlanCompletesDependenciesWithoutWriting(t *testing.T) {
	env := setupTestEnvironment(t)
	plan, err := BuildSetupPlan([]byte(`{}`), SetupRequest{Modules: []string{"route"}}, env)
	if err != nil {
		t.Fatal(err)
	}
	for _, module := range []string{"outbounds", "rule_sets", "dns", "route"} {
		if !slices.Contains(plan.Modules, module) {
			t.Errorf("missing dependency %q", module)
		}
	}
	if plan.Config["dns"] == nil || plan.Config["route"].(map[string]any)["final"] != "proxy" {
		t.Fatalf("incomplete default policy: %#v", plan.Config)
	}
	if _, err := os.Stat(env.DataDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview wrote to the data directory: %v", err)
	}
	if len(plan.CurrentConfig) != 0 || plan.SourceHash == "" {
		t.Fatal("preview must preserve the source and identify it")
	}
}

func TestSetupPlanPreservesConfiguredModules(t *testing.T) {
	body := []byte(`{"log":{"level":"debug"},"dns":{"servers":[{"type":"https","tag":"custom","server":"1.1.1.1"}],"final":"custom"},"inbounds":[{"type":"mixed","tag":"custom-in","listen":"127.0.0.1","listen_port":2080}]}`)
	plan, err := BuildSetupPlan(body, SetupRequest{}, setupTestEnvironment(t))
	if err != nil {
		t.Fatal(err)
	}
	if plan.Config["log"].(map[string]any)["level"] != "debug" {
		t.Fatal("custom log settings were overwritten")
	}
	if plan.Config["dns"].(map[string]any)["final"] != "custom" {
		t.Fatal("existing DNS policy was replaced")
	}
	before, err := json.Marshal(plan.CurrentConfig)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.CurrentConfig["inbounds"].([]any)) != 1 || len(before) == 0 {
		t.Fatal("building the plan changed the source object")
	}
}

func TestSetupPlanContainerListenerAndStoppedState(t *testing.T) {
	env := setupTestEnvironment(t)
	env.Capabilities.Container = true
	env.Capabilities.DefaultListen = "0.0.0.0"
	plan, err := BuildSetupPlan(nil, SetupRequest{Modules: []string{"inbounds"}, InboundMode: "proxy"}, env)
	if err != nil {
		t.Fatal(err)
	}
	inbounds := plan.Config["inbounds"].([]any)
	if inbounds[0].(map[string]any)["listen"] != "0.0.0.0" || len(inbounds) != 1 {
		t.Fatalf("container listener is not ready for publication: %#v", inbounds)
	}
	if plan.WillRestart {
		t.Fatal("a stopped kernel must remain stopped when saving")
	}
}

func TestSetupPlanRequiresExplicitInvalidConfigReset(t *testing.T) {
	env := setupTestEnvironment(t)
	if _, err := BuildSetupPlan([]byte(`broken JSON`), SetupRequest{}, env); err == nil {
		t.Fatal("invalid configuration cannot be silently replaced")
	}
	plan, err := BuildSetupPlan([]byte(`broken JSON`), SetupRequest{ResetInvalid: true}, env)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(plan.Warnings, "config_reset") {
		t.Fatal("reset must be visible before applying")
	}
}

func TestSetupPlanRejectsUnsupportedSelections(t *testing.T) {
	env := setupTestEnvironment(t)
	for _, tt := range []struct {
		name string
		req  SetupRequest
	}{
		{name: "unknown module", req: SetupRequest{Modules: []string{"unknown"}}},
		{name: "unsupported TUN", req: SetupRequest{Modules: []string{"inbounds"}, InboundMode: "tun"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := BuildSetupPlan([]byte(`{}`), tt.req, env); err == nil {
				t.Fatal("expected an actionable setup error")
			}
		})
	}
}

func TestSetupPlanExplicitlyRebuildsSemanticErrors(t *testing.T) {
	body := []byte(`{"log":{"level":"debug"},"inbounds":[{"type":"not-a-protocol","tag":"broken"}]}`)
	plan, err := BuildSetupPlan(body, SetupRequest{ResetInvalid: true}, setupTestEnvironment(t))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(plan.Warnings, "config_reset") {
		t.Fatal("semantic reset must be visible and rebuild from defaults")
	}
	if objectValue(plan.CurrentConfig["log"])["level"] != "debug" || plan.CurrentConfig["inbounds"] == nil {
		t.Fatal("preview must retain the original configuration to show removed custom values")
	}
	if objectValue(plan.Config["log"])["level"] != "info" {
		t.Fatal("the candidate must use the rebuilt defaults")
	}
	if _, err := encodeInitialConfig(plan.Config); err != nil {
		t.Fatalf("rebuilt configuration is invalid: %v", err)
	}
}

func TestSetupPlanResetFlagPreservesValidUserConfiguration(t *testing.T) {
	plan, err := BuildSetupPlan([]byte(`{"log":{"level":"debug"}}`), SetupRequest{ResetInvalid: true}, setupTestEnvironment(t))
	if err != nil {
		t.Fatal(err)
	}
	if plan.Config["log"].(map[string]any)["level"] != "debug" || slices.Contains(plan.Warnings, "config_reset") {
		t.Fatal("reset flag must not discard valid user configuration")
	}
}

func TestSetupPlanTUNWarningMatchesSelectedAddressFamily(t *testing.T) {
	for _, tt := range []struct {
		name      string
		available bool
		reason    string
		choice    string
		ipv4Only  bool
	}{
		{name: "known dual stack", available: true, choice: "auto"},
		{name: "explicit IPv4", available: true, choice: "off", ipv4Only: true},
		{name: "disabled IPv6", reason: NetworkIPv6Disabled, choice: "auto", ipv4Only: true},
		{name: "unknown explicitly enabled", reason: NetworkIPv6Unknown, choice: "on"},
		{name: "unknown explicitly disabled", reason: NetworkIPv6Unknown, choice: "off", ipv4Only: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTestEnvironment(t)
			env.Capabilities.TUNAvailable = true
			env.Capabilities.IPv6Available = tt.available
			env.Capabilities.IPv6Reason = tt.reason
			request := SetupRequest{Modules: []string{"inbounds"}, InboundMode: "tun", IPv6: tt.choice}
			plan, err := BuildSetupPlan(nil, request, env)
			if err != nil {
				t.Fatal(err)
			}
			if slices.Contains(plan.Warnings, "tun_ipv4_only") != tt.ipv4Only {
				t.Fatalf("IPv4-only warning must match the selected address family: %v", plan.Warnings)
			}
		})
	}
}
