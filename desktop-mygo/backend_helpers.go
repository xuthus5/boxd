package main

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func trafficTotalsFrom(values map[string]any) *trafficTotals {
	total := &trafficTotals{}
	switch value := values["upload_bytes"].(type) {
	case int64:
		total.Up = value
	case int:
		total.Up = int64(value)
	case float64:
		total.Up = int64(value)
	}
	switch value := values["download_bytes"].(type) {
	case int64:
		total.Down = value
	case int:
		total.Down = int64(value)
	case float64:
		total.Down = int64(value)
	}
	return total
}

func connectionViewsFrom(values map[string]any) []connectionView {
	list, _ := values["list"].([]core.TrafficConn)
	out := make([]connectionView, 0, len(list))
	now := time.Now()
	for _, conn := range list {
		out = append(out, connectionView{
			ID:       conn.ID,
			Target:   conn.Target,
			Outbound: conn.Outbound,
			Rule:     conn.Rule,
			Upload:   conn.Upload,
			Download: conn.Download,
			Duration: max(0, now.Sub(conn.Start)).Round(time.Second).String(),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func nodeViewsFrom(nodes []model.Outbound, results map[string]map[string]model.TestResult) []nodeView {
	out := make([]nodeView, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, nodeViewFrom(node, "import", "", results))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Tag < out[j].Tag })
	return out
}

func nodeViewFrom(node model.Outbound, source, sourceName string, results map[string]map[string]model.TestResult) nodeView {
	return nodeView{
		Tag: node.Tag, Type: node.Type, Server: node.Server, Port: node.Port,
		Test: nodeTestSummary(results, node.Tag), Source: source, SourceName: sourceName, Config: node.Raw,
	}
}

func nodeTestSummary(results map[string]map[string]model.TestResult, tag string) string {
	var latest model.TestResult
	found := false
	for key, byType := range results {
		if key != tag && !strings.HasPrefix(key, tag+"_") {
			continue
		}
		for _, result := range byType {
			if !found || result.Timestamp.After(latest.Timestamp) {
				latest, found = result, true
			}
		}
	}
	if !found {
		return "Not tested"
	}
	if !latest.Success {
		return "Failed"
	}
	return strconv.FormatFloat(latest.LatencyMs, 'f', 0, 64) + " ms"
}

func sortedLogs(entries []core.LogEntry) []core.LogEntry {
	out := append([]core.LogEntry(nil), entries...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out
}
