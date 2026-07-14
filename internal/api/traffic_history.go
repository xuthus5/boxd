package api

import (
	"sync"
	"time"
)

const (
	defaultTrafficHistoryLimit   = 600
	defaultTrafficSampleInterval = time.Second
)

type TrafficHistoryPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	UploadBytes   int64     `json:"upload_bytes"`
	DownloadBytes int64     `json:"download_bytes"`
}

type trafficHistoryBuffer struct {
	mu     sync.RWMutex
	points []TrafficHistoryPoint
	next   int
	full   bool
	limit  int
}

func newTrafficHistoryBuffer(limit int) *trafficHistoryBuffer {
	if limit <= 0 {
		limit = defaultTrafficHistoryLimit
	}

	return &trafficHistoryBuffer{
		points: make([]TrafficHistoryPoint, limit),
		limit:  limit,
	}
}

func (b *trafficHistoryBuffer) add(point TrafficHistoryPoint) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.points[b.next] = point
	b.next = (b.next + 1) % b.limit
	if b.next == 0 {
		b.full = true
	}
}

func (b *trafficHistoryBuffer) snapshot() []TrafficHistoryPoint {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.full && b.next == 0 {
		return []TrafficHistoryPoint{}
	}

	if !b.full {
		result := make([]TrafficHistoryPoint, b.next)
		copy(result, b.points[:b.next])
		return result
	}

	result := make([]TrafficHistoryPoint, 0, b.limit)
	result = append(result, b.points[b.next:]...)
	result = append(result, b.points[:b.next]...)
	return result
}
