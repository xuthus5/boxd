package core

import (
	"runtime/debug"
	"strings"
	"unicode"

	"github.com/xuthus5/boxd/internal/model"
)

type kernelBuildFeatures struct {
	known  bool
	goos   string
	goarch string
	cgo    bool
	tags   map[string]bool
}

func currentKernelBuildFeatures() kernelBuildFeatures {
	info, loaded := debug.ReadBuildInfo()
	if !loaded {
		return kernelBuildFeatures{}
	}
	return kernelBuildFeaturesFromInfo(info)
}

func kernelBuildFeaturesFromInfo(info *debug.BuildInfo) kernelBuildFeatures {
	features := kernelBuildFeatures{tags: make(map[string]bool)}
	if info == nil {
		return features
	}
	settings := make(map[string]string, len(info.Settings))
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	features.goos, features.goarch = settings["GOOS"], settings["GOARCH"]
	features.cgo = settings["CGO_ENABLED"] == "1"
	cgoKnown := features.cgo || settings["CGO_ENABLED"] == "0"
	for _, tag := range strings.FieldsFunc(settings["-tags"], func(value rune) bool { return value == ',' || unicode.IsSpace(value) }) {
		features.tags[tag] = true
	}
	features.known = features.goos != "" && features.goarch != "" && cgoKnown
	return features
}

func checkConfigBuildFeatures(report *model.ConfigDiagnostics, cfg map[string]any, features kernelBuildFeatures) {
	requirements := configBuildRequirements(cfg)
	if len(requirements) == 0 {
		return
	}
	if !features.known {
		addDiagnostic(report, "build_features_unknown", model.ConfigDiagnosticSeverityWarning, "config", "", "")
		return
	}
	for _, requirement := range requirements {
		if !features.supports(requirement) {
			addDiagnostic(report, "build_feature_unavailable", model.ConfigDiagnosticSeverityError,
				requirement.path, requirement.feature, requirement.tag)
		}
	}
}

func (features kernelBuildFeatures) supports(requirement configBuildRequirement) bool {
	if requirement.tag != "" && !features.tags[requirement.tag] {
		return false
	}
	apple := features.goos == "darwin" || features.goos == "ios"
	switch requirement.feature {
	case "apple":
		return apple && features.cgo
	case "usbip-client", "usbip-server":
		return features.supportsUSBIP()
	case "ccm":
		return !apple || features.cgo
	case "naive":
		return features.supportsNaive()
	default:
		return true
	}
}

func (features kernelBuildFeatures) supportsUSBIP() bool {
	return features.goos == "linux" || features.goos == "windows" || features.goos == "darwin" && features.cgo
}

func (features kernelBuildFeatures) supportsNaive() bool {
	puregoArch := features.goarch == "amd64" || features.goarch == "arm64"
	switch features.goos {
	case "windows":
		return features.tags["with_purego"] && puregoArch
	case "linux":
		return features.supportsLinuxNaive(puregoArch)
	case "darwin", "ios", "android":
		return features.cgo
	default:
		return false
	}
}

func (features kernelBuildFeatures) supportsLinuxNaive(puregoArch bool) bool {
	return features.cgo || features.tags["with_purego"] && puregoArch && !features.tags["with_musl"]
}
