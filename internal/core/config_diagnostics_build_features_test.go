package core

import (
	"context"
	"encoding/json"
	"runtime/debug"
	"strings"
	"testing"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"

	"github.com/xuthus5/boxd/internal/model"
)

func TestKernelBuildFeaturesReadActualSettings(t *testing.T) {
	if kernelBuildFeaturesFromInfo(nil).known {
		t.Fatal("missing build info must stay unknown")
	}
	settings := []debug.BuildSetting{
		{Key: "GOOS", Value: "linux"}, {Key: "GOARCH", Value: "amd64"},
		{Key: "CGO_ENABLED", Value: "0"}, {Key: "-tags", Value: "with_quic,with_utls with_gvisor\twith_dhcp"},
		{Key: "vcs.revision", Value: "unrelated"},
	}
	features := kernelBuildFeaturesFromInfo(&debug.BuildInfo{Settings: settings})
	if !features.known || features.cgo || len(features.tags) != 4 || !features.tags["with_quic"] {
		t.Fatalf("unexpected build capability snapshot: %+v", features)
	}
	for _, value := range []string{"", "unexpected"} {
		settings[2].Value = value
		if kernelBuildFeaturesFromInfo(&debug.BuildInfo{Settings: settings}).known {
			t.Fatal("unknown CGO metadata must not be treated as a release build")
		}
	}
	settings[2].Value = "1"
	if !kernelBuildFeaturesFromInfo(&debug.BuildInfo{Settings: settings}).cgo {
		t.Fatal("CGO-enabled build was not recognized")
	}
}

func TestConfigBuildFeaturesRequireOptionalTypeTags(t *testing.T) {
	for key, tag := range configTypeBuildTags {
		t.Run(key, func(t *testing.T) {
			section, feature, _ := strings.Cut(key, "/")
			entry := map[string]any{"type": feature, "system": true}
			cfg := buildTypeConfig(section, entry)
			report := newConfigDiagnostics()
			features := testKernelBuild("linux", "", true)
			checkConfigBuildFeatures(&report, cfg, features)
			requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
				code: "build_feature_unavailable", path: section + "[0].type", severity: model.ConfigDiagnosticSeverityError,
			})
			features.tags[tag] = true
			report = newConfigDiagnostics()
			checkConfigBuildFeatures(&report, cfg, features)
			if len(report.Issues) != 0 {
				t.Fatalf("compiled feature rejected: %+v", report.Issues)
			}
		})
	}
}

func TestConfigBuildFeaturesDoNotInventUnknownCapabilities(t *testing.T) {
	report := newConfigDiagnostics()
	checkConfigBuildFeatures(&report, buildTypeConfig("outbounds", map[string]any{"type": "naive"}), kernelBuildFeatures{})
	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "build_features_unknown", path: "config", severity: model.ConfigDiagnosticSeverityWarning,
	})
	requireNoConfigDiagnostic(t, report.Issues, "build_feature_unavailable", "")
	report = newConfigDiagnostics()
	checkConfigBuildFeatures(&report, buildTypeConfig("outbounds", map[string]any{"type": "direct"}), kernelBuildFeatures{})
	if len(report.Issues) != 0 {
		t.Fatal("ordinary types do not require optional build features")
	}
}

func TestAnalyzeConfigReportsNaiveBuildStub(t *testing.T) {
	const body = `{"outbounds":[{"type":"naive","tag":"n","server":"127.0.0.1","server_port":443,"username":"user","password":"hidden-build-secret","tls":{"enabled":true}}]}`
	report := AnalyzeConfig([]byte(body))
	features := currentKernelBuildFeatures()
	if !features.known || features.tags["with_naive_outbound"] {
		return
	}
	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "build_feature_unavailable", path: "outbounds[0].type", severity: model.ConfigDiagnosticSeverityError,
	})
	ctx := include.Context(context.Background())
	var options option.Options
	if err := options.UnmarshalJSONContext(ctx, []byte(body)); err != nil {
		t.Fatal(err)
	}
	instance, err := box.New(box.Options{Context: ctx, Options: options})
	if err == nil {
		_ = instance.Close()
		t.Fatal("native Naive stub unexpectedly constructed an outbound")
	}
	if !strings.Contains(err.Error(), "naive outbound is not included") {
		t.Fatalf("unexpected native feature rejection: %v", err)
	}
	encoded, err := json.Marshal(report)
	if err != nil || strings.Contains(string(encoded), "hidden-build-secret") {
		t.Fatalf("build diagnostic exposed credentials: %v", err)
	}
}

func buildTypeConfig(section string, entry map[string]any) map[string]any {
	if section == "dns.servers" {
		return map[string]any{"dns": map[string]any{"servers": []any{entry}}}
	}
	return map[string]any{section: []any{entry}}
}

func testKernelBuild(goos, tags string, cgo bool) kernelBuildFeatures {
	features := kernelBuildFeatures{known: true, goos: goos, goarch: "amd64", cgo: cgo, tags: map[string]bool{}}
	for _, tag := range strings.Fields(tags) {
		features.tags[tag] = true
	}
	return features
}
