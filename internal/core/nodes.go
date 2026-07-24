package core

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

var nodeBucket = []byte("imported_nodes")

var resultBucket = []byte("test_results")

var (
	ErrNodeNotFound    = errors.New("node not found")
	ErrNodeTagConflict = errors.New("node tag already exists")
)

type NodeManager struct {
	db *bbolt.DB
}

func NewNodeManager(db *bbolt.DB) *NodeManager {
	_ = db.Update(func(tx *bbolt.Tx) error {
		for _, bkt := range [][]byte{nodeBucket, resultBucket, historyBucket} {
			if _, err := tx.CreateBucketIfNotExists(bkt); err != nil {
				return err
			}
		}
		return nil
	})
	return &NodeManager{db: db}
}

func (m *NodeManager) List() []model.Outbound {
	var nodes []model.Outbound
	_ = m.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(nodeBucket)
		_ = b.ForEach(func(k, v []byte) error {
			var n model.Outbound
			if err := json.Unmarshal(v, &n); err == nil {
				nodes = append(nodes, n)
			}
			return nil
		})
		return nil
	})
	return nodes
}

func (m *NodeManager) Add(outbound model.Outbound) error {
	return m.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(nodeBucket)
		data, err := json.Marshal(outbound)
		if err != nil {
			return err
		}
		return b.Put([]byte(outbound.Tag), data)
	})
}

func (m *NodeManager) Delete(tag string) error {
	return m.db.Update(func(tx *bbolt.Tx) error {
		if b := tx.Bucket(resultBucket); b != nil {
			if err := deleteResultEntries(b, tag); err != nil {
				return err
			}
		}
		if b := tx.Bucket(historyBucket); b != nil {
			if err := b.Delete([]byte(tag)); err != nil {
				return err
			}
		}
		if b := tx.Bucket(nodeBucket); b != nil {
			return b.Delete([]byte(tag))
		}
		return nil
	})
}

func (m *NodeManager) Update(oldTag string, outbound model.Outbound) error {
	data, err := json.Marshal(outbound)
	if err != nil {
		return err
	}
	return m.db.Update(func(tx *bbolt.Tx) error {
		nodes := tx.Bucket(nodeBucket)
		if nodes == nil || nodes.Get([]byte(oldTag)) == nil {
			return ErrNodeNotFound
		}
		if oldTag != outbound.Tag && nodes.Get([]byte(outbound.Tag)) != nil {
			return ErrNodeTagConflict
		}
		if oldTag != outbound.Tag {
			if err := moveResultEntries(tx.Bucket(resultBucket), oldTag, outbound.Tag); err != nil {
				return err
			}
			if err := moveHistoryEntry(tx.Bucket(historyBucket), oldTag, outbound.Tag); err != nil {
				return err
			}
			if err := nodes.Delete([]byte(oldTag)); err != nil {
				return err
			}
		}
		return nodes.Put([]byte(outbound.Tag), data)
	})
}

func (m *NodeManager) Get(tag string) *model.Outbound {
	var node *model.Outbound
	_ = m.db.View(func(tx *bbolt.Tx) error {
		data := tx.Bucket(nodeBucket).Get([]byte(tag))
		if data == nil {
			return nil
		}
		var n model.Outbound
		if err := json.Unmarshal(data, &n); err == nil {
			node = &n
		}
		return nil
	})
	return node
}

// ---- Test result persistence ----

type StoredResult struct {
	Results map[string]model.TestResult `json:"results"`
}

func (m *NodeManager) SaveTestResult(key string, result model.TestResult) error {
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now().UTC()
	}
	err := m.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(resultBucket)
		var sr StoredResult
		if data := b.Get([]byte(key)); data != nil {
			if err := json.Unmarshal(data, &sr); err != nil {
				sr.Results = nil
			}
		}
		if sr.Results == nil {
			sr.Results = make(map[string]model.TestResult)
		}
		sr.Results[result.TestType] = result
		data, err := json.Marshal(sr)
		if err != nil {
			return err
		}
		return b.Put([]byte(key), data)
	})
	if err != nil {
		return err
	}
	tag := result.Tag
	if tag == "" {
		// key format tag_testType
		if i := strings.LastIndex(key, "_"); i > 0 {
			tag = key[:i]
		} else {
			tag = key
		}
	}
	_ = m.AppendTestHistory(tag, result.TestType, model.LatencyPoint{
		Timestamp: result.Timestamp,
		Success:   result.Success,
		LatencyMs: result.LatencyMs,
		Error:     result.Error,
	})
	return nil
}

func (m *NodeManager) GetAllTestResults() map[string]map[string]model.TestResult {
	all := make(map[string]map[string]model.TestResult)
	_ = m.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(resultBucket)
		_ = b.ForEach(func(k, v []byte) error {
			var sr StoredResult
			if err := json.Unmarshal(v, &sr); err == nil && sr.Results != nil {
				all[string(k)] = sr.Results
			}
			return nil
		})
		return nil
	})
	return all
}
