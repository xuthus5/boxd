package core

import (
	"encoding/json"
	"strings"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

var historyBucket = []byte("test_history")

const defaultLatencyHistoryLimit = 48

// historySeries stores capped probe samples per test type under a node tag.
type historySeries struct {
	ByType map[string][]model.LatencyPoint `json:"by_type"`
}

// AppendTestHistory records one probe sample for tag/testType, capped to limit.
func (m *NodeManager) AppendTestHistory(tag, testType string, point model.LatencyPoint) error {
	tag = strings.TrimSpace(tag)
	testType = strings.TrimSpace(testType)
	if tag == "" || testType == "" {
		return nil
	}
	if point.Timestamp.IsZero() {
		point.Timestamp = time.Now().UTC()
	}
	return m.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(historyBucket)
		if err != nil {
			return err
		}
		series := historySeries{ByType: map[string][]model.LatencyPoint{}}
		if data := b.Get([]byte(tag)); data != nil {
			if unmarshalErr := json.Unmarshal(data, &series); unmarshalErr != nil {
				series.ByType = map[string][]model.LatencyPoint{}
			}
		}
		if series.ByType == nil {
			series.ByType = map[string][]model.LatencyPoint{}
		}
		points := append(series.ByType[testType], point)
		if len(points) > defaultLatencyHistoryLimit {
			points = points[len(points)-defaultLatencyHistoryLimit:]
		}
		series.ByType[testType] = points
		payload, err := json.Marshal(series)
		if err != nil {
			return err
		}
		return b.Put([]byte(tag), payload)
	})
}

// GetTestHistory returns probe history for one tag (empty map if none).
func (m *NodeManager) GetTestHistory(tag string) map[string][]model.LatencyPoint {
	out := map[string][]model.LatencyPoint{}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return out
	}
	_ = m.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(historyBucket)
		if b == nil {
			return nil
		}
		data := b.Get([]byte(tag))
		if data == nil {
			return nil
		}
		var series historySeries
		if err := json.Unmarshal(data, &series); err != nil || series.ByType == nil {
			return nil
		}
		for k, v := range series.ByType {
			cp := make([]model.LatencyPoint, len(v))
			copy(cp, v)
			out[k] = cp
		}
		return nil
	})
	return out
}

// GetAllTestHistory returns probe history keyed by node tag.
func (m *NodeManager) GetAllTestHistory() map[string]map[string][]model.LatencyPoint {
	all := make(map[string]map[string][]model.LatencyPoint)
	_ = m.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(historyBucket)
		if b == nil {
			return nil
		}
		_ = b.ForEach(func(k, v []byte) error {
			var series historySeries
			if err := json.Unmarshal(v, &series); err != nil || series.ByType == nil {
				return nil
			}
			copied := make(map[string][]model.LatencyPoint, len(series.ByType))
			for typ, points := range series.ByType {
				cp := make([]model.LatencyPoint, len(points))
				copy(cp, points)
				copied[typ] = cp
			}
			all[string(k)] = copied
			return nil
		})
		return nil
	})
	return all
}

// DeleteTestHistory removes history for a node tag.
func (m *NodeManager) DeleteTestHistory(tag string) error {
	return m.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(historyBucket)
		if b == nil {
			return nil
		}
		return b.Delete([]byte(tag))
	})
}
