package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func TestRuntimeBackendUnavailable(t *testing.T) {
	var b *runtimeBackend
	ctx := context.Background()
	if _, err := b.Snapshot(ctx); !errors.Is(err, errRuntimeUnavailable) {
		t.Fatalf("Snapshot error = %v", err)
	}
	for name, err := range map[string]error{
		"start":            b.StartKernel(ctx),
		"stop":             b.StopKernel(ctx),
		"restart":          b.RestartKernel(ctx),
		"close-connection": b.CloseConnection(ctx, 1),
		"refresh-sub":      b.RefreshSubscription(ctx, "1"),
		"kernel-autostart": b.SetKernelAutostart(ctx, true),
		"preferences":      b.SetUIPreferences(ctx, model.UIPreferences{}),
		"test-url":         b.SetTestURL(ctx, "https://example.test"),
		"import":           b.ImportLink(ctx, "vless://example"),
	} {
		if !errors.Is(err, errRuntimeUnavailable) {
			t.Errorf("%s error = %v", name, err)
		}
	}
}

func TestTrafficTotalsFrom(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]any
		want trafficTotals
	}{
		{name: "int64", in: map[string]any{"upload_bytes": int64(12), "download_bytes": int64(34)}, want: trafficTotals{Up: 12, Down: 34}},
		{name: "int", in: map[string]any{"upload_bytes": 5, "download_bytes": 8}, want: trafficTotals{Up: 5, Down: 8}},
		{name: "float", in: map[string]any{"upload_bytes": 1.9, "download_bytes": 2.9}, want: trafficTotals{Up: 1, Down: 2}},
		{name: "missing", in: map[string]any{}, want: trafficTotals{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trafficTotalsFrom(tt.in)
			if got == nil || *got != tt.want {
				t.Fatalf("trafficTotalsFrom = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNodeTestSummary(t *testing.T) {
	older := time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)
	newer := older.Add(time.Minute)
	results := map[string]map[string]model.TestResult{
		"proxy": {
			"tcp": {Tag: "proxy", Success: true, LatencyMs: 22, Timestamp: older},
			"udp": {Tag: "proxy", Success: false, Timestamp: newer},
		},
		"other": {
			"tcp": {Tag: "other", Success: true, LatencyMs: 1, Timestamp: newer},
		},
	}
	if got := nodeTestSummary(results, "proxy"); got != "Failed" {
		t.Fatalf("proxy summary = %q", got)
	}
	if got := nodeTestSummary(results, "missing"); got != "Not tested" {
		t.Fatalf("missing summary = %q", got)
	}
	results["proxy"]["udp"] = model.TestResult{Tag: "proxy", Success: true, LatencyMs: 35, Timestamp: newer}
	if got := nodeTestSummary(results, "proxy"); got != "35 ms" {
		t.Fatalf("success summary = %q", got)
	}
}

func TestNodeViewsAndConnections(t *testing.T) {
	nodes := nodeViewsFrom([]model.Outbound{
		{Tag: "z", Type: "trojan", Server: "z.example", Port: 443},
		{Tag: "a", Type: "vless", Server: "a.example", Port: 8443},
	}, nil)
	if len(nodes) != 2 || nodes[0].Tag != "a" || nodes[1].Tag != "z" {
		t.Fatalf("sorted nodes = %+v", nodes)
	}

	now := time.Now()
	connections := connectionViewsFrom(map[string]any{"list": []core.TrafficConn{
		{ID: 2, Target: "b:80", Upload: 4, Download: 8, Start: now.Add(-2 * time.Second)},
		{ID: 1, Target: "a:80", Upload: 1, Download: 2, Start: now.Add(-3 * time.Second)},
	}})
	if len(connections) != 2 || connections[0].ID != 1 || connections[1].ID != 2 {
		t.Fatalf("sorted connections = %+v", connections)
	}
	if connections[0].Duration == "" || totalConnectionUpload(connections) != 5 || totalConnectionDownload(connections) != 10 {
		t.Fatalf("connection details = %+v", connections)
	}
}

func TestSortedLogs(t *testing.T) {
	later := core.LogEntry{Message: "later", Timestamp: time.Unix(20, 0)}
	earlier := core.LogEntry{Message: "earlier", Timestamp: time.Unix(10, 0)}
	got := sortedLogs([]core.LogEntry{later, earlier})
	if len(got) != 2 || got[0].Message != "earlier" || got[1].Message != "later" {
		t.Fatalf("sorted logs = %+v", got)
	}
	got[0].Message = "changed"
	if later.Message != "later" {
		t.Fatal("sortedLogs mutated its input slice")
	}
}
