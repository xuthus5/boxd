package singbox

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"

	singboxconstant "github.com/sagernet/sing-box/constant"

	"github.com/xuthus5/boxd/kernel"
)

const (
	// ModulePath identifies the engine module whose version is governed here.
	ModulePath = "github.com/sagernet/sing-box"
	// SupportedMajor and SupportedMinor define the only compatible engine line.
	SupportedMajor = 1
	SupportedMinor = 14
	// SupportedSeries is the human-readable compatibility line.
	SupportedSeries = "v1.14.x"
)

// VersionInfo separates the module version selected by Go from an optional
// version injected into sing-box through linker flags.
type VersionInfo struct {
	ModuleVersion    string
	InjectedVersion  string
	EffectiveVersion string
	Series           string
}

// CurrentVersion reports the engine version embedded in the running binary.
// Go build information is authoritative; linker-injected sing-box versions are
// accepted only when present.
func CurrentVersion() VersionInfo {
	moduleVersion := readModuleVersion()
	injected := normalizeVersion(singboxconstant.Version)
	effective := injected
	if effective == "" {
		effective = moduleVersion
	}
	return VersionInfo{
		ModuleVersion:    moduleVersion,
		InjectedVersion:  injected,
		EffectiveVersion: effective,
		Series:           SupportedSeries,
	}
}

// CheckCompatibility fails closed when the effective engine version cannot be
// determined or falls outside the supported minor line.
func CheckCompatibility() error {
	info := CurrentVersion()
	if err := validateVersion(info.EffectiveVersion); err != nil {
		return kernel.NewError(
			kernel.ErrCodeEngineIncompatible,
			"driver.singbox.version",
			fmt.Sprintf("sing-box %s is not compatible with %s", displayVersion(info.EffectiveVersion), SupportedSeries),
		).WithHint(err.Error())
	}
	return nil
}

func readModuleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	if info.Main.Path == ModulePath {
		return normalizeVersion(info.Main.Version)
	}
	for _, dependency := range info.Deps {
		if dependency.Path != ModulePath {
			continue
		}
		if dependency.Replace != nil {
			return normalizeVersion(dependency.Replace.Version)
		}
		return normalizeVersion(dependency.Version)
	}
	return ""
}

func validateVersion(version string) error {
	version = normalizeVersion(version)
	if version == "" {
		return fmt.Errorf("the sing-box module version is unavailable in build information")
	}
	core := strings.TrimPrefix(version, "v")
	if index := strings.IndexAny(core, "-+"); index >= 0 {
		core = core[:index]
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return fmt.Errorf("version %q is not a full semantic version", version)
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	_, patchErr := strconv.Atoi(parts[2])
	if majorErr != nil || minorErr != nil || patchErr != nil {
		return fmt.Errorf("version %q has a non-numeric component", version)
	}
	if major != SupportedMajor || minor != SupportedMinor {
		return fmt.Errorf("supported engine line is %s; update the driver compatibility contract before using %s", SupportedSeries, version)
	}
	return nil
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	switch strings.ToLower(version) {
	case "", "unknown", "devel", "(devel)":
		return ""
	}
	if version[0] >= '0' && version[0] <= '9' {
		version = "v" + version
	}
	return version
}

func displayVersion(version string) string {
	if version == "" {
		return "version unknown"
	}
	return version
}
