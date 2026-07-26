package core

import (
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

func TestAnalyzeConfigReportsOutboundGroupRuntimeErrors(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "outbounds":[
    {"type":"direct","tag":"direct"},
    {"type":"direct","tag":"other"},
    {"type":"selector","tag":"selector","outbounds":["direct"],"default":"other"},
    {"type":"urltest","tag":"empty","outbounds":[]}
  ]
}`))

	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "invalid_group_default", severity: model.ConfigDiagnosticSeverityError,
		path: "outbounds[2].default", value: "other",
	})
	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "empty_group", severity: model.ConfigDiagnosticSeverityError,
		path: "outbounds[3].outbounds", value: "urltest",
	})
}

func TestAnalyzeConfigReportsOutboundDependencyCycles(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "outbounds":[
    {"type":"selector","tag":"group-a","outbounds":["group-b"]},
    {"type":"urltest","tag":"group-b","outbounds":["group-a"]},
    {"type":"direct","tag":"detour-a","detour":"detour-b"},
    {"type":"direct","tag":"detour-b","detour":"detour-a"}
  ]
}`))

	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "outbound_dependency_cycle", severity: model.ConfigDiagnosticSeverityError,
		path: "outbounds[1].outbounds[0]", value: "group-a",
	})
	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "outbound_dependency_cycle", severity: model.ConfigDiagnosticSeverityError,
		path: "outbounds[3].detour", value: "detour-a",
	})
}
