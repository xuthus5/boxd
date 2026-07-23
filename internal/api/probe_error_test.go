package api

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"testing"
)

func TestClassifyProbeError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		msg  string
		err  error
		want string
	}{
		{name: "unavailable", msg: "test service not available", want: ProbeErrorUnavailable},
		{name: "invalid", msg: "invalid server address", want: ProbeErrorInvalidInput},
		{name: "timeout msg", msg: "i/o timeout", want: ProbeErrorTimeout},
		{name: "deadline", err: context.DeadlineExceeded, want: ProbeErrorTimeout},
		{name: "network op", err: &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}, want: ProbeErrorNetwork},
		{name: "dns error", err: &net.DNSError{Err: "no such host", Name: "x"}, want: ProbeErrorNetwork},
		{name: "no response", msg: "delay test failed: no response", want: ProbeErrorNoResponse},
		{name: "unsupported", msg: `dns type "local" is not probeable`, want: ProbeErrorUnsupported},
		{name: "rcode", msg: "dns rcode SERVFAIL", want: ProbeErrorDNSRcode},
		{name: "empty", msg: "empty dns response", want: ProbeErrorEmpty},
		{name: "ping", msg: "ping failed: 100% packet loss", want: ProbeErrorNetwork},
		{name: "unknown", msg: "something weird", want: ProbeErrorUnknown},
		{name: "empty msg", msg: "", want: ProbeErrorUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyProbeError(tt.msg, tt.err)
			if got != tt.want {
				t.Fatalf("classifyProbeError(%q, %v) = %q, want %q", tt.msg, tt.err, got, tt.want)
			}
		})
	}
}

func TestFailedTestResult(t *testing.T) {
	t.Parallel()
	got := failedTestResult("test service not available", nil)
	if got.Success || got.Error != "test service not available" || got.ErrorCode != ProbeErrorUnavailable {
		t.Fatalf("failedTestResult = %+v", got)
	}
	fromErr := failedTestResult("", fmt.Errorf("connection refused"))
	if fromErr.ErrorCode != ProbeErrorNetwork || fromErr.Error != "connection refused" {
		t.Fatalf("fromErr = %+v", fromErr)
	}
}

func TestFailedDNSProbeResult(t *testing.T) {
	t.Parallel()
	base := DNSProbeResult{Tag: "cf", Domain: "example.com"}
	got := failedDNSProbeResult(base, "empty dns response", nil)
	if got.Success || got.ErrorCode != ProbeErrorEmpty || got.Tag != "cf" {
		t.Fatalf("failedDNSProbeResult = %+v", got)
	}
}
