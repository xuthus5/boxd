package singbox

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	singboxconstant "github.com/sagernet/sing-box/constant"

	"github.com/xuthus5/boxd/kernel"
)

func TestValidateVersionAcceptsPatchReleasesInSupportedLine(t *testing.T) {
	tests := []struct {
		version string
		valid   bool
	}{
		{version: "v1.14.4", valid: true},
		{version: "1.14.4", valid: true},
		{version: "v1.14.10", valid: true},
		{version: "v1.14.4-beta.1", valid: true},
		{version: "v1.14.4+boxd.1", valid: true},
		{version: "v1.13.9", valid: false},
		{version: "v1.15.0", valid: false},
		{version: "v2.14.0", valid: false},
		{version: "v1.14", valid: false},
		{version: "v1.14.x", valid: false},
		{version: "v1.14.4.1", valid: false},
		{version: "dev", valid: false},
		{version: "", valid: false},
	}
	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			err := validateVersion(test.version)
			if test.valid && err != nil {
				t.Fatalf("validateVersion(%q) error = %v", test.version, err)
			}
			if !test.valid && err == nil {
				t.Fatalf("validateVersion(%q) succeeded", test.version)
			}
		})
	}
}

func TestSupportedSeriesMatchesNumericConstraints(t *testing.T) {
	if got := fmt.Sprintf("v%d.%d.x", SupportedMajor, SupportedMinor); got != SupportedSeries {
		t.Fatalf("SupportedSeries = %q, numeric constraints describe %q", SupportedSeries, got)
	}
}

func TestCheckCompatibilityRejectsInjectedVersionOutsideLine(t *testing.T) {
	previous := singboxconstant.Version
	singboxconstant.Version = "v1.15.0"
	t.Cleanup(func() { singboxconstant.Version = previous })

	err := CheckCompatibility()
	var kernelError *kernel.Error
	if !errors.As(err, &kernelError) || kernelError.Code != kernel.ErrCodeEngineIncompatible {
		t.Fatalf("CheckCompatibility() error = %v, want incompatible engine", err)
	}
}

func TestCurrentBuildUsesSupportedEngineLine(t *testing.T) {
	info := CurrentVersion()
	if info.EffectiveVersion == "" {
		t.Fatal("sing-box version is missing from build information")
	}
	if !strings.HasPrefix(info.EffectiveVersion, "v1.14.") {
		t.Fatalf("effective version = %q, want v1.14.x", info.EffectiveVersion)
	}
	if err := CheckCompatibility(); err != nil {
		t.Fatalf("CheckCompatibility() error = %v", err)
	}
}
