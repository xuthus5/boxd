package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/xuthus5/boxd/internal/core"
)

// statsInstance 抽取 StatsHandler 所需的运行时能力，便于测试注入。
type statsInstance interface {
	TrafficTracker() *core.TrafficTracker
	CloseConnection(id int64) bool
	CloseAllConnections() int
	CloseConnectionsByOutbound(outbound string) int
	CloseConnectionsByRule(rule string) int
	CloseConnectionsByProcess(process string) int
	CloseConnectionsByIDs(ids []int64) int
}

type StatsHandler struct {
	kernelLogWriter *core.LogWriter
	appLogWriter    *core.LogWriter
	instance        statsInstance
	history         *trafficHistoryBuffer
	now             func() time.Time
	stopOnce        sync.Once
	stopCh          chan struct{}
}

const sseHeartbeatInterval = 15 * time.Second

func NewStatsHandler(kernelLogWriter, appLogWriter *core.LogWriter, instance statsInstance) *StatsHandler {
	handler := &StatsHandler{
		kernelLogWriter: kernelLogWriter,
		appLogWriter:    appLogWriter,
		instance:        instance,
		history:         newTrafficHistoryBuffer(defaultTrafficHistoryLimit),
		now:             time.Now,
		stopCh:          make(chan struct{}),
	}

	handler.recordTrafficHistoryPoint()
	if instance != nil {
		go handler.runTrafficSampler(defaultTrafficSampleInterval)
	}

	return handler
}

func writeSSE(w http.ResponseWriter, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		slog.Error("sse marshal error", "err", err)
		return
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", raw)
	if err != nil {
		slog.Error("sse write error", "err", err)
	}
}

func writeSSEComment(w http.ResponseWriter, comment string) {
	_, err := fmt.Fprintf(w, ": %s\n\n", comment)
	if err != nil {
		slog.Error("sse comment write error", "err", err)
	}
}

func prepareSSE(w http.ResponseWriter) (http.Flusher, bool) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return nil, false
	}
	writeSSEComment(w, "connected")
	flusher.Flush()
	return flusher, true
}

func (h *StatsHandler) getTraffic() (up, down int64) {
	if h.instance != nil && h.instance.TrafficTracker() != nil {
		return h.instance.TrafficTracker().Total()
	}
	return 0, 0
}

func (h *StatsHandler) getConns() (int, []core.TrafficConn) {
	if h.instance != nil && h.instance.TrafficTracker() != nil {
		list := h.instance.TrafficTracker().Connections()
		return len(list), list
	}
	return 0, nil
}

func (h *StatsHandler) runTrafficSampler(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.recordTrafficHistoryPoint()
		}
	}
}

func (h *StatsHandler) stopTrafficSampler() {
	h.stopOnce.Do(func() {
		close(h.stopCh)
	})
}

func (h *StatsHandler) recordTrafficHistoryPoint() {
	if h.history == nil {
		return
	}

	up, down := h.getTraffic()
	h.history.add(TrafficHistoryPoint{
		Timestamp:     h.now().UTC(),
		UploadBytes:   up,
		DownloadBytes: down,
	})
}

func (h *StatsHandler) TrafficHistory(w http.ResponseWriter, r *http.Request) {
	points := []TrafficHistoryPoint{}
	if h.history != nil {
		points = h.history.snapshot()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"points": points,
	})
}

func (h *StatsHandler) TrafficSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := prepareSSE(w)
	if !ok {
		return
	}

	ctx := r.Context()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			up, down := h.getTraffic()
			writeSSE(w, map[string]any{
				"upload_bytes":   up,
				"download_bytes": down,
				"timestamp":      time.Now().Format(time.RFC3339),
			})
			flusher.Flush()
		}
	}
}

func (h *StatsHandler) streamLogs(w http.ResponseWriter, r *http.Request, lw *core.LogWriter) {
	flusher, ok := prepareSSE(w)
	if !ok {
		return
	}

	ctx := r.Context()
	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()
	var ch <-chan core.LogEntry
	var subscriptionID int
	if lw != nil {
		var recent []core.LogEntry
		recent, ch, subscriptionID = lw.SnapshotAndSubscribe()
		defer lw.Unsubscribe(subscriptionID)
		for _, entry := range recent {
			writeSSE(w, entry)
		}
		flusher.Flush()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			writeSSEComment(w, "keep-alive")
			flusher.Flush()
		case entry, open := <-ch:
			if !open {
				return
			}
			writeSSE(w, entry)
			flusher.Flush()
		}
	}
}

// LogsSSE 输出 sing-box 内核日志流。
func (h *StatsHandler) LogsSSE(w http.ResponseWriter, r *http.Request) {
	h.streamLogs(w, r, h.kernelLogWriter)
}

// AppLogsSSE 输出 boxd 自身应用日志流。
func (h *StatsHandler) AppLogsSSE(w http.ResponseWriter, r *http.Request) {
	h.streamLogs(w, r, h.appLogWriter)
}

func (h *StatsHandler) ConnectionsSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := prepareSSE(w)
	if !ok {
		return
	}

	ctx := r.Context()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, list := h.getConns()
			writeSSE(w, map[string]any{
				"active_connections": count,
				"list":               list,
			})
			flusher.Flush()
		}
	}
}
