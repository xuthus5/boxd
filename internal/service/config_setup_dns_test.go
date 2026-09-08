package service

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
)

func TestConfigSetupDNSUnsafeProxyReturns400WithoutWriting(t *testing.T) {
	const original = `{"outbounds":[{"type":"direct","tag":"direct"},{"type":"selector","tag":"proxy","outbounds":["direct"]}],"route":{"final":"proxy"}}`
	for _, action := range []string{"preview", "apply"} {
		t.Run(action, func(t *testing.T) {
			svc := newTestService(t)
			if err := os.WriteFile(svc.Deps.ConfigPath, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			probe := &configReloadProbe{running: true}
			svc.Config().instance = probe
			request := core.SetupRequest{Modules: []string{"dns"}, SourceHash: core.ConfigContentHash([]byte(original))}
			var err error
			if action == "preview" {
				_, err = svc.Config().PreviewSetup(t.Context(), request)
			} else {
				_, err = svc.Config().ApplySetup(t.Context(), request)
			}
			assertSetupDomainError(t, err, "400/config_invalid")
			var domain *DomainError
			if !errors.As(err, &domain) || !strings.Contains(domain.Message, "remove direct fallback") {
				t.Fatalf("DNS prerequisite error lacks an actionable hint: %v", err)
			}
			if probe.reloads != 0 || probe.restarts != 0 {
				t.Fatalf("rejected DNS setup changed runtime state: %+v", probe)
			}
			assertServiceSetupSource(t, svc.Deps.ConfigPath, original)
		})
	}
}
