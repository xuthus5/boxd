//go:build linux

package core

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

type capabilityCloser struct {
	err    error
	closed bool
}

func (c *capabilityCloser) Close() error { c.closed = true; return c.err }

func TestLinuxTUNCapabilityRequiresEffectivePermissionAndDevice(t *testing.T) {
	for _, tt := range []struct {
		name                    string
		admin                   bool
		adminErr, openErr, shut error
		reason                  string
	}{
		{name: "available", admin: true},
		{name: "no capability", reason: NetworkTUNPermission},
		{name: "capability check failed", adminErr: os.ErrPermission, reason: NetworkTUNUnknown},
		{name: "device missing", admin: true, openErr: os.ErrNotExist, reason: NetworkTUNDevice},
		{name: "device not permitted", admin: true, openErr: os.ErrPermission, reason: NetworkTUNDevice},
		{name: "close failed", admin: true, shut: io.ErrClosedPipe, reason: NetworkTUNUnknown},
	} {
		t.Run(tt.name, func(t *testing.T) {
			closer := &capabilityCloser{err: tt.shut}
			available, reason := linuxTUNCapability(func() (bool, error) { return tt.admin, tt.adminErr },
				func() (io.Closer, error) { return closer, tt.openErr })
			if available != (tt.reason == "") || reason != tt.reason {
				t.Fatalf("unexpected TUN capability: %v/%s", available, reason)
			}
			if tt.admin && tt.adminErr == nil && tt.openErr == nil && !closer.closed {
				t.Fatal("TUN probe leaked its device handle")
			}
		})
	}
}

func TestLinuxIPv6CapabilityChecksDefaultAndAllNamespaces(t *testing.T) {
	for _, tt := range []struct {
		name, all, defaults, reason string
		readErr, socketErr          error
	}{
		{name: "enabled without public address", all: "0\n", defaults: "0"},
		{name: "disabled globally", all: "1", defaults: "0", reason: NetworkIPv6Disabled},
		{name: "disabled for new interfaces", all: "0", defaults: "1", reason: NetworkIPv6Disabled},
		{name: "kernel IPv6 missing", readErr: os.ErrNotExist, socketErr: unix.EAFNOSUPPORT, reason: NetworkIPv6Unsupported},
		{name: "IPv6 protocol missing", socketErr: unix.EPROTONOSUPPORT, reason: NetworkIPv6Unsupported},
		{name: "sysctl hidden by container", readErr: os.ErrNotExist, reason: NetworkIPv6Unknown},
		{name: "sysctl unreadable", readErr: os.ErrPermission, reason: NetworkIPv6Unknown},
		{name: "malformed sysctl", all: "unknown", reason: NetworkIPv6Unknown},
		{name: "socket unavailable", all: "0", defaults: "0", socketErr: io.ErrClosedPipe, reason: NetworkIPv6Unknown},
	} {
		t.Run(tt.name, func(t *testing.T) {
			read := func(path string) ([]byte, error) {
				if strings.Contains(path, "/default/") {
					return []byte(tt.defaults), tt.readErr
				}
				return []byte(tt.all), tt.readErr
			}
			available, reason := linuxIPv6Capability(read, func() error { return tt.socketErr })
			if available != (tt.reason == "") || reason != tt.reason {
				t.Fatalf("want reason %s, got %v/%s", tt.reason, available, reason)
			}
		})
	}
}

func TestLinuxTUNProbeOnlyChecksAndClosesExistingDevice(t *testing.T) {
	admin, err := linuxNetworkAdmin()
	if err != nil {
		t.Skipf("capability information unavailable: %v", err)
	}
	device, openErr := openLinuxTUN()
	if openErr == nil {
		if err := device.Close(); err != nil {
			t.Fatal(err)
		}
	} else if !errors.Is(openErr, os.ErrNotExist) && !errors.Is(openErr, os.ErrPermission) {
		t.Logf("device unavailable: %v", openErr)
	}
	if got := bootstrapTUNAvailable(); got != (admin && openErr == nil) {
		t.Fatalf("unexpected bootstrap capability %v", got)
	}
}
