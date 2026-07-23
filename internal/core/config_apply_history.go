package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

const (
	// DefaultConfigApplyHistoryLimit caps retained config apply events.
	DefaultConfigApplyHistoryLimit = 30

	configApplyStatusApplied    = "applied"
	configApplyStatusRolledBack = "rolled_back"
)

var configApplyHistoryBucket = []byte("config_apply_history")

var configApplyHistoryKey = []byte("events")

// ConfigApplyHistoryManager persists recent config apply/reload events.
type ConfigApplyHistoryManager struct {
	db    *bbolt.DB
	limit int
}

// NewConfigApplyHistoryManager creates a history store with the default cap.
func NewConfigApplyHistoryManager(db *bbolt.DB) *ConfigApplyHistoryManager {
	return &ConfigApplyHistoryManager{db: db, limit: DefaultConfigApplyHistoryLimit}
}

// ConfigBodyHash returns a short sha256 fingerprint of config bytes.
func ConfigBodyHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// NewConfigApplyEvent builds a timeline event for a successful write attempt.
func NewConfigApplyEvent(source, status string, body []byte, applyErr error) model.ConfigApplyEvent {
	event := model.ConfigApplyEvent{
		ID:        newConfigApplyID(),
		Source:    strings.TrimSpace(source),
		Status:    normalizeConfigApplyStatus(status),
		Hash:      ConfigBodyHash(body),
		Size:      len(body),
		AppliedAt: time.Now().UTC(),
	}
	if applyErr != nil {
		event.Error = strings.TrimSpace(applyErr.Error())
	}
	if event.Source == "" {
		event.Source = "unknown"
	}
	return event
}

func normalizeConfigApplyStatus(status string) string {
	if strings.TrimSpace(status) == configApplyStatusRolledBack {
		return configApplyStatusRolledBack
	}
	return configApplyStatusApplied
}

func newConfigApplyID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(buf[:])
}

// Append records one apply event (newest first) and trims to the configured limit.
func (m *ConfigApplyHistoryManager) Append(event model.ConfigApplyEvent) error {
	if m == nil || m.db == nil {
		return nil
	}
	if event.ID == "" {
		event.ID = newConfigApplyID()
	}
	if event.AppliedAt.IsZero() {
		event.AppliedAt = time.Now().UTC()
	}
	event.Status = normalizeConfigApplyStatus(event.Status)
	event.Source = strings.TrimSpace(event.Source)
	if event.Source == "" {
		event.Source = "unknown"
	}
	limit := m.limit
	if limit <= 0 {
		limit = DefaultConfigApplyHistoryLimit
	}
	return m.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(configApplyHistoryBucket)
		if err != nil {
			return err
		}
		events := decodeConfigApplyEvents(bucket.Get(configApplyHistoryKey))
		events = append([]model.ConfigApplyEvent{event}, events...)
		if len(events) > limit {
			events = events[:limit]
		}
		payload, err := json.Marshal(events)
		if err != nil {
			return err
		}
		return bucket.Put(configApplyHistoryKey, payload)
	})
}

// List returns recent apply events newest-first (up to limit).
func (m *ConfigApplyHistoryManager) List(limit int) ([]model.ConfigApplyEvent, error) {
	if m == nil || m.db == nil {
		return []model.ConfigApplyEvent{}, nil
	}
	if limit <= 0 {
		limit = m.limit
	}
	if limit <= 0 {
		limit = DefaultConfigApplyHistoryLimit
	}
	var events []model.ConfigApplyEvent
	err := m.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(configApplyHistoryBucket)
		if bucket == nil {
			return nil
		}
		events = decodeConfigApplyEvents(bucket.Get(configApplyHistoryKey))
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(events) > limit {
		events = events[:limit]
	}
	if events == nil {
		events = []model.ConfigApplyEvent{}
	}
	return events, nil
}

func decodeConfigApplyEvents(data []byte) []model.ConfigApplyEvent {
	if len(data) == 0 {
		return nil
	}
	var events []model.ConfigApplyEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil
	}
	return events
}
