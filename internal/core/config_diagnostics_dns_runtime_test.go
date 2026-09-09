package core

import (
	"context"
	"errors"
	"os"
	"testing"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

func TestDNSResponseDiagnosticsMatchSingBox114Runtime(t *testing.T) {
	for _, test := range dnsResponseDiagnosticCases {
		t.Run(test.name, func(t *testing.T) {
			err := diagnosticKernelStartError(t, dnsResponseTestConfig(test.rules))
			if test.warning && err != nil {
				t.Fatalf("warning-only configuration must remain supported in 1.14: %v", err)
			}
			if !test.warning && err == nil {
				t.Fatalf("error diagnostic %s disagrees with 1.14 startup", test.code)
			}
		})
	}
}

func diagnosticKernelStartError(t *testing.T, body string) error {
	t.Helper()
	ctx, cancel := context.WithCancel(include.Context(t.Context()))
	defer cancel()
	var options option.Options
	if err := options.UnmarshalJSONContext(ctx, []byte(body)); err != nil {
		return err
	}
	options.Log = &option.LogOptions{Disabled: true}
	instance, err := newRealBox(box.Options{Context: ctx, Options: options})
	if err != nil {
		return err
	}
	startErr := instance.Start()
	if closeErr := instance.Close(); closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
		t.Errorf("close diagnostic runtime: %v", closeErr)
	}
	return startErr
}
