package core

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
)

func TestClassifySubscriptionRefreshError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{"nil", nil, ""},
		{"already classified", newSubscriptionRefreshError(SubRefreshEmpty, "empty", 0), SubRefreshEmpty},
		{"not found", errors.New("subscription not found: 1"), SubRefreshNotFound},
		{"timeout text", errors.New("i/o timeout"), SubRefreshTimeout},
		{"deadline", context.DeadlineExceeded, SubRefreshTimeout},
		{"network text", errors.New("dial tcp: no such host"), SubRefreshNetwork},
		{"invalid url", errors.New("parse \"://bad\": unsupported protocol scheme \"\""), SubRefreshInvalidURL},
		{"unknown", errors.New("weird failure"), SubRefreshUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifySubscriptionRefreshError(tt.err)
			if tt.err == nil {
				if got != nil {
					t.Fatalf("got = %#v", got)
				}
				return
			}
			if got == nil || got.Code != tt.code {
				t.Fatalf("got = %#v want %s", got, tt.code)
			}
		})
	}
}

func TestClassifyHTTPStatus(t *testing.T) {
	if got := classifyHTTPStatus(401); got.Code != SubRefreshUnauthorized {
		t.Fatalf("401 = %#v", got)
	}
	if got := classifyHTTPStatus(403); got.Code != SubRefreshForbidden {
		t.Fatalf("403 = %#v", got)
	}
	if got := classifyHTTPStatus(502); got.Code != SubRefreshHTTP || got.Status != 502 {
		t.Fatalf("502 = %#v", got)
	}
}

func TestIsNetworkAndTimeoutHelpers(t *testing.T) {
	if !isTimeoutError(context.DeadlineExceeded) {
		t.Fatal("deadline should be timeout")
	}
	if !isNetworkError(&net.OpError{Err: syscall.ECONNREFUSED}) {
		t.Fatal("op error should be network")
	}
	if !isNetworkError(syscall.ECONNRESET) {
		t.Fatal("ECONNRESET should be network")
	}
}
