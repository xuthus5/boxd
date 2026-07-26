package core

import (
	"context"
	"errors"
	"testing"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	M "github.com/sagernet/sing/common/metadata"
)

func TestDNSOutboundBootstrapCyclesMatchRuntimeDialFailures(t *testing.T) {
	for _, body := range []string{
		explicitDNSOutboundBootstrapCycleConfig,
		defaultDNSOutboundBootstrapCycleConfig,
		detouredDNSOutboundBootstrapCycleConfig,
	} {
		if err := singBoxOutboundDialError(t, body, "proxy"); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("runtime error = %v, want context deadline exceeded", err)
		}
	}
}

func singBoxOutboundDialError(t *testing.T, body, tag string) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(include.Context(context.Background()), 100*time.Millisecond)
	defer cancel()
	var options option.Options
	if err := options.UnmarshalJSONContext(ctx, []byte(body)); err != nil {
		t.Fatal(err)
	}
	instance, err := box.New(box.Options{Context: ctx, Options: options})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := instance.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	if err := instance.Start(); err != nil {
		t.Fatal(err)
	}
	outbound, loaded := instance.Outbound().Outbound(tag)
	if !loaded {
		t.Fatalf("outbound %q not found", tag)
	}
	conn, dialErr := outbound.DialContext(ctx, "tcp", M.ParseSocksaddr("example.com:80"))
	if conn == nil {
		return dialErr
	}
	return errors.Join(dialErr, conn.Close())
}
