package core

import (
	"encoding/json"
	"testing"
)

func TestKernelBuildFeaturesRespectPlatformAndCGO(t *testing.T) {
	for _, test := range []struct {
		name, feature, goos, tags string
		cgo, supported            bool
	}{
		{"Linux Naive unavailable", "naive", "linux", "with_naive_outbound", false, false},
		{"Linux purego", "naive", "linux", "with_naive_outbound with_purego", false, true},
		{"Linux musl needs CGO", "naive", "linux", "with_naive_outbound with_purego with_musl", false, false},
		{"Linux musl CGO", "naive", "linux", "with_naive_outbound with_musl", true, true},
		{"Windows purego", "naive", "windows", "with_naive_outbound with_purego", false, true},
		{"Windows missing purego", "naive", "windows", "with_naive_outbound", true, false},
		{"Apple Naive needs CGO", "naive", "darwin", "with_naive_outbound", false, false},
		{"Apple Naive CGO", "naive", "darwin", "with_naive_outbound", true, true},
		{"Android CGO", "naive", "android", "with_naive_outbound", true, true},
		{"Other Naive platform", "naive", "freebsd", "with_naive_outbound", true, false},
		{"USBIP Linux", "usbip-client", "linux", "with_usbip", false, true},
		{"USBIP Windows", "usbip-server", "windows", "with_usbip", false, true},
		{"USBIP macOS no CGO", "usbip-client", "darwin", "with_usbip", false, false},
		{"USBIP macOS CGO", "usbip-server", "darwin", "with_usbip", true, true},
		{"USBIP iOS unsupported", "usbip-client", "ios", "with_usbip", true, false},
		{"CCM macOS no CGO", "ccm", "darwin", "with_ccm", false, false},
		{"CCM macOS CGO", "ccm", "darwin", "with_ccm", true, true},
		{"Apple engine Linux", "apple", "linux", "", true, false},
		{"Apple engine macOS", "apple", "darwin", "", true, true},
		{"Apple engine iOS", "apple", "ios", "", true, true},
		{"Apple engine no CGO", "apple", "darwin", "", false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			features := testKernelBuild(test.goos, test.tags, test.cgo)
			if features.supports(configBuildRequirement{feature: test.feature}) != test.supported {
				t.Fatalf("unexpected platform capability: %+v", test)
			}
		})
	}
	features := testKernelBuild("linux", "with_naive_outbound with_purego", false)
	features.goarch = "386"
	if features.supportsNaive() {
		t.Fatal("purego Naive only supports amd64 and arm64")
	}
}

func TestConfigBuildRequirementsInspectOnlyTypedFields(t *testing.T) {
	const body = `{
 "inbounds":[{"type":"naive","network":"tcp"},{"type":"naive","network":"udp"}],
 "outbounds":[{"type":"vless","transport":{"type":"quic"},"tls":{"enabled":true,"utls":{"enabled":true},"engine":"apple"}},
 {"type":"direct","tls":{"enabled":false,"utls":{"enabled":true}},"headers":{"tls":{"enabled":true,"utls":{"enabled":true}}}}],
 "endpoints":[{"type":"openconnect","system":false}],
 "http_clients":[{"tag":"apple","engine":"apple"},{"tag":"go","engine":"go"}],
 "services":[{"type":"api","tls":{"enabled":true,"certificate_provider":{"type":"acme","http_client":{"engine":"apple"}}}}]
}`
	var cfg map[string]any
	if err := json.Unmarshal([]byte(body), &cfg); err != nil {
		t.Fatal(err)
	}
	requirements := configBuildRequirements(cfg)
	paths := make(map[string]bool)
	for _, requirement := range requirements {
		paths[requirement.path] = true
	}
	expected := []string{
		"inbounds[1].network", "outbounds[0].transport.type", "outbounds[0].tls.utls.enabled",
		"outbounds[0].tls.engine", "endpoints[0].type", "endpoints[0].system",
		"http_clients[0].engine", "services[0].tls.certificate_provider.type",
		"services[0].tls.certificate_provider.http_client.engine",
	}
	if len(paths) != len(expected) {
		t.Fatalf("unexpected feature paths: %v", paths)
	}
	for _, path := range expected {
		if !paths[path] {
			t.Errorf("missing feature requirement at %s", path)
		}
	}
}
