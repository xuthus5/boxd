package core

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestSetupDNSUnsafeProxyReturnsActionableConfigurationError(t *testing.T) {
	body := []byte(`{"outbounds":[{"type":"direct","tag":"direct"},{"type":"selector","tag":"proxy","outbounds":["direct"]}],"route":{"final":"proxy"}}`)
	original := bytes.Clone(body)
	env := setupTestEnvironment(t)
	_, err := BuildSetupPlan(body, SetupRequest{Modules: []string{"dns"}}, env)
	var setupErr *SetupError
	if !errors.As(err, &setupErr) || setupErr.Code != "config_invalid" || !strings.Contains(setupErr.Message, "remove direct fallback") {
		t.Fatalf("unsafe DNS proxy needs an actionable configuration error: %v", err)
	}
	if !bytes.Equal(original, body) {
		t.Fatal("DNS setup rewrote the user's proxy configuration")
	}
	if _, err := os.Stat(env.DataDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rejected DNS setup created assets: %v", err)
	}
}

func TestSetupDNSInstallErrorMapsOnlyDNSPreconditions(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		hint string
	}{
		{name: "unsafe proxy", err: fmt.Errorf("proxy choice: %w", ErrDNSProxyUnsafe), hint: "remove direct fallback"},
		{name: "missing proxy", err: fmt.Errorf("proxy choice: %w", ErrDNSProxyRequired), hint: "install proxy outbound defaults"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var setupErr *SetupError
			if err := setupDNSInstallError(tt.err); !errors.As(err, &setupErr) || setupErr.Code != "config_invalid" || !strings.Contains(setupErr.Message, tt.hint) {
				t.Fatalf("unexpected DNS prerequisite error: %v", err)
			}
		})
	}
	if err := setupDNSInstallError(nil); err != nil {
		t.Fatalf("successful installation returned an error: %v", err)
	}
	unrelated := errors.New("unrelated failure")
	if err := setupDNSInstallError(unrelated); !errors.Is(err, unrelated) {
		t.Fatalf("unrelated error was misclassified: %v", err)
	}
}
