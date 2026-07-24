package core

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

func TestClassifyRuleSetUpdateError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		msg  string
		err  error
		want string
	}{
		{name: "not updatable", err: ErrRuleSetNotUpdatable, want: RuleSetErrorNotUpdatable},
		{name: "unsupported local", msg: "custom local rule-set files are not auto-updated", want: RuleSetErrorUnsupported},
		{name: "empty url", msg: "remote rule-set url is empty", want: RuleSetErrorInvalidURL},
		{name: "blocked url", err: errSubscriptionURLBlocked, want: RuleSetErrorBlockedURL},
		{name: "invalid url", err: errSubscriptionURLInvalid, want: RuleSetErrorInvalidURL},
		{name: "too large", err: ErrRuleSetContentTooLarge, want: RuleSetErrorTooLarge},
		{name: "http", msg: "unexpected status 418", want: RuleSetErrorHTTP},
		{name: "empty", msg: "empty rule-set body", want: RuleSetErrorEmpty},
		{name: "timeout", err: context.DeadlineExceeded, want: RuleSetErrorTimeout},
		{name: "network", err: &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}, want: RuleSetErrorNetwork},
		{name: "cache disabled", err: ErrRuleSetCacheDisabled, want: RuleSetErrorCache},
		{name: "unknown", msg: "something else", want: RuleSetErrorUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyRuleSetUpdateError(tt.msg, tt.err)
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestFailRuleSetResult(t *testing.T) {
	t.Parallel()
	base := model.RuleSetUpdateResult{Tag: "geo", Type: "remote"}
	got := failRuleSetResult(base, "unexpected status 500", nil)
	if got.OK || got.ErrorCode != RuleSetErrorHTTP || got.Tag != "geo" {
		t.Fatalf("%+v", got)
	}
	fromErr := failRuleSetResult(base, "", fmt.Errorf("connection refused"))
	if fromErr.ErrorCode != RuleSetErrorNetwork {
		t.Fatalf("%+v", fromErr)
	}
}
