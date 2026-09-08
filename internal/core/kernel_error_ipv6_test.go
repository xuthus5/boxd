package core

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

const windowsIPv6AddressError = "start inbound/tun[tun-in]: configure tun interface: " +
	"set ipv6 address: element not found"

const linuxIPv6AddressError = "start inbound/tun[tun-in]: configure tun interface: " +
	"add address fdfe:dcba:9876::1/126: permission denied"

func TestClassifyKernelIPv6AddressErrors(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		msg  string
		err  error
	}{
		{name: "windows adapter error", msg: windowsIPv6AddressError},
		{name: "windows missing resource", msg: windowsIPv6AddressError, err: os.ErrNotExist},
		{name: "windows case insensitive", msg: "SET IPV6 ADDRESS: The system cannot find the file specified"},
		{name: "linux permission", msg: linuxIPv6AddressError, err: os.ErrPermission},
		{name: "wrapped error only", err: fmt.Errorf("add address 2001:db8::1/64: %w", os.ErrPermission)},
		{name: "maximum prefix", msg: "add address ::1/128: address family not supported"},
		{name: "zero prefix", msg: "add address ::/0: address family not supported"},
		{name: "mixed error chain", msg: "add address 192.0.2.1/24: denied; " + linuxIPv6AddressError},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := ClassifyKernelError(test.msg, test.err); got != "ipv6_unavailable" {
				t.Fatalf("want ipv6_unavailable, got %q for message %q and error %v", got, test.msg, test.err)
			}
		})
	}
}

func TestClassifyKernelIPv6RequiresAddressContext(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		msg  string
		err  error
		want string
	}{
		{name: "plain permission", msg: "permission denied", want: KernelErrorPermission},
		{name: "config path contains ipv6", err: fmt.Errorf("open ipv6.json: %w", os.ErrNotExist), want: KernelErrorConfigMissing},
		{name: "ipv4 prefix", msg: "add address 192.0.2.1/24: permission denied", want: KernelErrorPermission},
		{name: "invalid ipv6", msg: "add address 2001:db8:::1/64: permission denied", want: KernelErrorPermission},
		{name: "out of range prefix", msg: "add address fdfe::1/129: permission denied", want: KernelErrorPermission},
		{name: "noncanonical prefix", msg: "add address fdfe::1/064: permission denied", want: KernelErrorPermission},
		{name: "missing prefix", msg: "add address fdfe::1: permission denied", want: KernelErrorPermission},
		{name: "empty address", msg: "add address ", err: os.ErrPermission, want: KernelErrorPermission},
		{name: "ipv6 listener", msg: "listen tcp [::]:1080: permission denied", want: KernelErrorPermission},
		{name: "unrelated element", msg: "element not found", want: KernelErrorUnknown},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := ClassifyKernelError(test.msg, test.err); got != test.want {
				t.Fatalf("want %q, got %q for message %q and error %v", test.want, got, test.msg, test.err)
			}
		})
	}
}

func TestSBInstanceIPv6ErrorDiagnostics(t *testing.T) {
	fixture := newSBReloadFixture(t)
	fixture.startErr = errors.New(windowsIPv6AddressError)
	if err := fixture.instance.Start(); !errors.Is(err, fixture.startErr) {
		t.Fatalf("want IPv6 setup error, got %v", err)
	}
	status := fixture.instance.Status()
	if status.LastErrorCode != "ipv6_unavailable" || status.LastError != windowsIPv6AddressError {
		t.Fatalf("IPv6 diagnostics must retain the original error and specific code: %#v", status)
	}
}

func TestConfigApplyEventIPv6ErrorDiagnostics(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("restart failed after config save: %w", errors.New(linuxIPv6AddressError))
	event := NewConfigApplyEvent("update", "rolled_back", []byte(`{}`), err)
	if event.ErrorCode != "ipv6_unavailable" || event.Error != err.Error() {
		t.Fatalf("apply history must retain the IPv6 error context: %#v", event)
	}
}
