package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

const (
	// DefaultConfigApplyHistoryLimit caps retained config apply events.
	DefaultConfigApplyHistoryLimit = 30
	// MaxConfigSnapshotBytes caps the size of one retained configuration snapshot.
	MaxConfigSnapshotBytes = 2 * 1024 * 1024

	configApplyStatusApplied        = model.ConfigApplyStatusApplied
	configApplyStatusRolledBack     = model.ConfigApplyStatusRolledBack
	configApplyStatusValidated      = model.ConfigApplyStatusValidated
	configApplyStatusValidateFailed = model.ConfigApplyStatusValidateFailed
)

var configApplyHistoryBucket = []byte("config_apply_history")

var configApplyHistoryKey = []byte("events")

var configApplySnapshotsBucket = []byte("config_apply_snapshots")

var ErrConfigSnapshotNotFound = errors.New("config snapshot not found")

// ConfigApplyHistoryManager persists recent config apply/reload events.
type ConfigApplyHistoryManager struct {
	db    *bbolt.DB
	limit int
}

type appendConfigApplyInput struct {
	event         model.ConfigApplyEvent
	body          []byte
	limit         int
	storeSnapshot bool
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
		event.ErrorCode = ClassifyKernelError(event.Error, applyErr)
	}
	if event.Source == "" {
		event.Source = "unknown"
	}
	return event
}

func normalizeConfigApplyStatus(status string) string {
	switch strings.TrimSpace(status) {
	case configApplyStatusRolledBack:
		return configApplyStatusRolledBack
	case configApplyStatusValidated:
		return configApplyStatusValidated
	case configApplyStatusValidateFailed:
		return configApplyStatusValidateFailed
	default:
		return configApplyStatusApplied
	}
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
	return m.append(event, nil)
}

// AppendSnapshot records an apply event and retains its successful config body.
func (m *ConfigApplyHistoryManager) AppendSnapshot(event model.ConfigApplyEvent, body []byte) error {
	return m.append(event, body)
}

func (m *ConfigApplyHistoryManager) append(event model.ConfigApplyEvent, body []byte) error {
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
	storeSnapshot := event.Status == configApplyStatusApplied && len(body) > 0 && len(body) <= MaxConfigSnapshotBytes
	event.Restorable = storeSnapshot
	limit := m.limit
	if limit <= 0 {
		limit = DefaultConfigApplyHistoryLimit
	}
	return m.db.Update(func(tx *bbolt.Tx) error {
		return appendConfigApplyEvent(tx, appendConfigApplyInput{
			event:         event,
			body:          body,
			limit:         limit,
			storeSnapshot: storeSnapshot,
		})
	})
}

func appendConfigApplyEvent(tx *bbolt.Tx, input appendConfigApplyInput) error {
	bucket, err := tx.CreateBucketIfNotExists(configApplyHistoryBucket)
	if err != nil {
		return err
	}
	events := append([]model.ConfigApplyEvent{input.event}, decodeConfigApplyEvents(bucket.Get(configApplyHistoryKey))...)
	if len(events) > input.limit {
		events = events[:input.limit]
	}
	if input.storeSnapshot {
		if err := storeConfigSnapshot(tx, input.event.ID, input.body); err != nil {
			return err
		}
	}
	payload, err := json.Marshal(events)
	if err != nil {
		return err
	}
	if err := bucket.Put(configApplyHistoryKey, payload); err != nil {
		return err
	}
	return trimConfigSnapshots(tx, events)
}

func storeConfigSnapshot(tx *bbolt.Tx, id string, body []byte) error {
	snapshots, err := tx.CreateBucketIfNotExists(configApplySnapshotsBucket)
	if err != nil {
		return err
	}
	return snapshots.Put([]byte(id), append([]byte(nil), body...))
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
		markRestorableEvents(tx, events)
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

// GetSnapshot returns a retained successful config body by event ID.
func (m *ConfigApplyHistoryManager) GetSnapshot(id string) ([]byte, error) {
	if m == nil || m.db == nil {
		return nil, ErrConfigSnapshotNotFound
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrConfigSnapshotNotFound
	}
	var snapshot []byte
	err := m.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(configApplySnapshotsBucket)
		if bucket == nil {
			return ErrConfigSnapshotNotFound
		}
		data := bucket.Get([]byte(id))
		if len(data) == 0 {
			return ErrConfigSnapshotNotFound
		}
		snapshot = append([]byte(nil), data...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func markRestorableEvents(tx *bbolt.Tx, events []model.ConfigApplyEvent) {
	snapshots := tx.Bucket(configApplySnapshotsBucket)
	for index := range events {
		events[index].Restorable = events[index].Status == configApplyStatusApplied && snapshots != nil && snapshots.Get([]byte(events[index].ID)) != nil
	}
}

func trimConfigSnapshots(tx *bbolt.Tx, events []model.ConfigApplyEvent) error {
	snapshots := tx.Bucket(configApplySnapshotsBucket)
	if snapshots == nil {
		return nil
	}
	retained := make(map[string]struct{}, len(events))
	for _, event := range events {
		if event.Status == configApplyStatusApplied && snapshots.Get([]byte(event.ID)) != nil {
			retained[event.ID] = struct{}{}
		}
	}
	var stale [][]byte
	cursor := snapshots.Cursor()
	for key, _ := cursor.First(); key != nil; key, _ = cursor.Next() {
		if _, ok := retained[string(key)]; !ok {
			stale = append(stale, append([]byte(nil), key...))
		}
	}
	for _, key := range stale {
		if err := snapshots.Delete(key); err != nil {
			return err
		}
	}
	return nil
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
