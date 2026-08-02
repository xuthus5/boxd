package service

import (
	"context"
	"strconv"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
)

type fakeStats struct {
	tracker  *core.TrafficTracker
	closeOne bool
}

func (f *fakeStats) TrafficTracker() *core.TrafficTracker { return f.tracker }
func (f *fakeStats) CloseConnection(int64) bool           { return f.closeOne }
func (f *fakeStats) CloseAllConnections() int             { return 3 }
func (f *fakeStats) CloseConnectionsByOutbound(string) int {
	return 2
}
func (f *fakeStats) CloseConnectionsByRule(string) int { return 1 }
func (f *fakeStats) CloseConnectionsByProcess(string) int {
	return 4
}
func (f *fakeStats) CloseConnectionsByIDs([]int64) int { return 5 }

func TestStatsTraffic(t *testing.T) {
	tracker := core.NewTrafficTracker()
	svc := NewStatsService(&fakeStats{tracker: tracker})
	result, err := svc.Traffic(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result["upload_bytes"] != int64(0) || result["download_bytes"] != int64(0) {
		t.Fatalf("result = %v", result)
	}
}

func TestStatsTrafficNilInstance(t *testing.T) {
	svc := NewStatsService(nil)
	result, err := svc.Traffic(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result["upload_bytes"] != int64(0) || result["download_bytes"] != int64(0) {
		t.Fatalf("result = %v", result)
	}
}

func TestStatsConnections(t *testing.T) {
	tracker := core.NewTrafficTracker()
	svc := NewStatsService(&fakeStats{tracker: tracker})
	result, err := svc.Connections(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result["active_connections"] != 0 {
		t.Fatalf("result = %v", result)
	}
}

func TestStatsConnectionsNilInstance(t *testing.T) {
	svc := NewStatsService(nil)
	result, err := svc.Connections(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result["active_connections"] != 0 {
		t.Fatalf("result = %v", result)
	}
}

func TestStatsCloseAll(t *testing.T) {
	svc := NewStatsService(&fakeStats{})
	result, err := svc.CloseConnections(context.Background(), ConnectionCloseFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if result["closed"] != 3 {
		t.Fatalf("result = %v", result)
	}
}

func TestStatsCloseByFilter(t *testing.T) {
	svc := NewStatsService(&fakeStats{})
	tests := []ConnectionCloseFilters{
		{Outbound: "p"},
		{Rule: "r"},
		{Process: "pr"},
		{IDs: []string{"1", "2"}},
	}
	for _, filters := range tests {
		if _, err := svc.CloseConnections(context.Background(), filters); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStatsCloseMultipleFiltersRejected(t *testing.T) {
	svc := NewStatsService(&fakeStats{})
	_, err := svc.CloseConnections(context.Background(), ConnectionCloseFilters{Outbound: "p", Rule: "r"})
	if err == nil {
		t.Fatal("expected error for multiple filters")
	}
}

func TestStatsCloseNilInstance(t *testing.T) {
	svc := NewStatsService(nil)
	if _, err := svc.CloseConnections(context.Background(), ConnectionCloseFilters{}); err == nil {
		t.Fatal("expected error for nil instance")
	}
}

func TestStatsCloseConnection(t *testing.T) {
	svc := NewStatsService(&fakeStats{closeOne: true})
	result, err := svc.CloseConnection(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if result["closed_id"] != 5 {
		t.Fatalf("result = %v", result)
	}
}

func TestStatsCloseConnectionNotFound(t *testing.T) {
	svc := NewStatsService(&fakeStats{closeOne: false})
	if _, err := svc.CloseConnection(context.Background(), 5); err == nil {
		t.Fatal("expected not found error")
	}
}

func TestStatsCloseConnectionNilInstance(t *testing.T) {
	svc := NewStatsService(nil)
	if _, err := svc.CloseConnection(context.Background(), 5); err == nil {
		t.Fatal("expected error for nil instance")
	}
}

func TestParseConnectionIDs(t *testing.T) {
	ids, err := ParseConnectionIDs("1, 2, 3")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 {
		t.Fatalf("ids = %v", ids)
	}
	ids, err = ParseConnectionIDs("")
	if err != nil || len(ids) != 0 {
		t.Fatalf("ids = %v err %v", ids, err)
	}
	ids, err = ParseConnectionIDs("1,1,2")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("ids = %v", ids)
	}
}

func TestParseConnectionIDsErrors(t *testing.T) {
	if _, err := ParseConnectionIDs("abc"); err == nil {
		t.Fatal("expected error for non-numeric")
	}
	if _, err := ParseConnectionIDs("-1"); err == nil {
		t.Fatal("expected error for negative")
	}
	big := make([]string, maxCloseConnectionIDs+1)
	raw := ""
	for i := range big {
		if i > 0 {
			raw += ","
		}
		raw += strconv.Itoa(i + 1)
	}
	if _, err := ParseConnectionIDs(raw); err == nil {
		t.Fatal("expected error for too many ids")
	}
}
