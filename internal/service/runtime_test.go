package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/core"
)

type fakeRuntime struct {
	groups       []core.OutboundGroupInfo
	selectErr    error
	urlTest      map[string]uint16
	urlTestErr   error
	flushDNSErr  error
	flushFakeErr error
	delay        uint16
	delayErr     error
	clash        core.ClashModeStatus
	clashErr     error
	setClashErr  error
}

func (f *fakeRuntime) OutboundGroups() []core.OutboundGroupInfo { return f.groups }
func (f *fakeRuntime) SelectOutbound(_, _ string) error         { return f.selectErr }
func (f *fakeRuntime) URLTestDelays(context.Context, string) (map[string]uint16, error) {
	return f.urlTest, f.urlTestErr
}
func (f *fakeRuntime) FlushDNS() error { return f.flushDNSErr }
func (f *fakeRuntime) FlushFakeIP() error {
	return f.flushFakeErr
}
func (f *fakeRuntime) OutboundDelay(context.Context, string, string, time.Duration) (uint16, error) {
	return f.delay, f.delayErr
}
func (f *fakeRuntime) ClashMode() (core.ClashModeStatus, error) { return f.clash, f.clashErr }
func (f *fakeRuntime) SetClashMode(string) (core.ClashModeStatus, error) {
	return f.clash, f.setClashErr
}

func TestRuntimeOutboundGroups(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{groups: []core.OutboundGroupInfo{{Tag: "g"}}})
	groups, err := svc.OutboundGroups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("groups = %d", len(groups))
	}
}

func TestRuntimeOutboundGroupsNilInstance(t *testing.T) {
	svc := NewRuntimeService(nil)
	if _, err := svc.OutboundGroups(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestRuntimeSelectOutbound(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{})
	tag, err := svc.SelectOutbound(context.Background(), "g", "n")
	if err != nil {
		t.Fatal(err)
	}
	if tag != "n" {
		t.Fatalf("tag = %q", tag)
	}
}

func TestRuntimeSelectOutboundMissingGroup(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{})
	if _, err := svc.SelectOutbound(context.Background(), "", "n"); err == nil {
		t.Fatal("expected error for empty group")
	}
	if _, err := svc.SelectOutbound(context.Background(), "g", ""); err == nil {
		t.Fatal("expected error for empty tag")
	}
}

func TestRuntimeSelectOutboundErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "not running", err: core.ErrNotRunning},
		{name: "group not found", err: core.ErrGroupNotFound},
		{name: "not selectable", err: core.ErrNotSelectable},
		{name: "tag not in group", err: core.ErrTagNotInGroup},
		{name: "other", err: errors.New("boom")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc := NewRuntimeService(&fakeRuntime{selectErr: test.err})
			if _, err := svc.SelectOutbound(context.Background(), "g", "n"); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestRuntimeURLTestDelays(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{urlTest: map[string]uint16{"a": 10}})
	delays, err := svc.URLTestDelays(context.Background(), "g")
	if err != nil {
		t.Fatal(err)
	}
	if delays["a"] != 10 {
		t.Fatalf("delays = %v", delays)
	}
}

func TestRuntimeURLTestDelaysErrors(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{urlTestErr: core.ErrNotRunning})
	if _, err := svc.URLTestDelays(context.Background(), "g"); err == nil {
		t.Fatal("expected error")
	}
	svc2 := NewRuntimeService(&fakeRuntime{})
	if _, err := svc2.URLTestDelays(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty group")
	}
}

func TestRuntimeFlushDNS(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{})
	if err := svc.FlushDNS(context.Background()); err != nil {
		t.Fatal(err)
	}
	svc2 := NewRuntimeService(&fakeRuntime{flushDNSErr: core.ErrNotRunning})
	if err := svc2.FlushDNS(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestRuntimeFlushFakeIP(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{})
	if err := svc.FlushFakeIP(context.Background()); err != nil {
		t.Fatal(err)
	}
	svc2 := NewRuntimeService(&fakeRuntime{flushFakeErr: core.ErrNotRunning})
	if err := svc2.FlushFakeIP(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestRuntimeOutboundDelay(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{delay: 50})
	delay, err := svc.OutboundDelay(context.Background(), "n", "", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if delay != 50 {
		t.Fatalf("delay = %d", delay)
	}
}

func TestRuntimeOutboundDelayMissingTag(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{})
	if _, err := svc.OutboundDelay(context.Background(), "", "", 5000); err == nil {
		t.Fatal("expected error for empty tag")
	}
}

func TestRuntimeOutboundDelayInvalidLink(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{})
	if _, err := svc.OutboundDelay(context.Background(), "n", "not a url", 5000); err == nil {
		t.Fatal("expected error for invalid link")
	}
}

func TestRuntimeOutboundDelayInvalidTimeout(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{delay: 10})
	delay, err := svc.OutboundDelay(context.Background(), "n", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if delay != 10 {
		t.Fatalf("delay = %d", delay)
	}
	svc2 := NewRuntimeService(&fakeRuntime{delay: 10})
	if _, err := svc2.OutboundDelay(context.Background(), "n", "", 70000); err == nil {
		t.Fatal("expected error for oversized timeout")
	}
}

func TestRuntimeOutboundDelayZeroDelay(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{})
	if _, err := svc.OutboundDelay(context.Background(), "n", "", 5000); err == nil {
		t.Fatal("expected error for zero delay")
	}
}

func TestRuntimeOutboundDelayNotRunning(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{delayErr: core.ErrNotRunning})
	if _, err := svc.OutboundDelay(context.Background(), "n", "", 5000); err == nil {
		t.Fatal("expected error")
	}
}

func TestRuntimeClashMode(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{clash: core.ClashModeStatus{Mode: "rule"}})
	status, err := svc.GetClashMode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Mode != "rule" {
		t.Fatalf("mode = %q", status.Mode)
	}
	status, err = svc.SetClashMode(context.Background(), "global")
	if err != nil {
		t.Fatal(err)
	}
	if status.Mode != "rule" {
		t.Fatalf("mode = %q", status.Mode)
	}
}

func TestRuntimeClashModeErrors(t *testing.T) {
	svc := NewRuntimeService(&fakeRuntime{clashErr: core.ErrNotRunning})
	if _, err := svc.GetClashMode(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	svc2 := NewRuntimeService(&fakeRuntime{setClashErr: core.ErrInvalidMode})
	if _, err := svc2.SetClashMode(context.Background(), "bad"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseTimeout(t *testing.T) {
	if got, err := ParseTimeout("", 10000); err != nil || got != 10000 {
		t.Fatalf("got %d err %v", got, err)
	}
	if got, err := ParseTimeout("5000", 10000); err != nil || got != 5000 {
		t.Fatalf("got %d err %v", got, err)
	}
	if _, err := ParseTimeout("abc", 10000); err == nil {
		t.Fatal("expected error")
	}
	if _, err := ParseTimeout("70000", 10000); err == nil {
		t.Fatal("expected error for oversized")
	}
}
