package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxui/internal/model"
)

var subBucket = []byte("subscriptions")

type SubscriptionManager struct {
	mu      sync.RWMutex
	db      *bbolt.DB
	dataDir string
}

var subscriptionHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
	},
}

func NewSubscriptionManager(db *bbolt.DB, dataDir string) *SubscriptionManager {
	_ = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(subBucket)
		return err
	})
	return &SubscriptionManager{
		db:      db,
		dataDir: dataDir,
	}
}

func (m *SubscriptionManager) DB() *bbolt.DB {
	return m.db
}

func (m *SubscriptionManager) List() []model.Subscription {
	m.mu.RLock()
	defer m.mu.RUnlock()

	subs := make([]model.Subscription, 0)
	_ = m.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subBucket)
		_ = b.ForEach(func(k, v []byte) error {
			var sub model.Subscription
			if err := json.Unmarshal(v, &sub); err == nil {
				subs = append(subs, sub)
			}
			return nil
		})
		return nil
	})
	return subs
}

func (m *SubscriptionManager) Create(name, url string, interval int) (*model.Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var sub model.Subscription
	err := m.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subBucket)
		id, err := b.NextSequence()
		if err != nil {
			return err
		}

		sub = model.Subscription{
			ID:          fmt.Sprintf("%d", id),
			Name:        name,
			URL:         url,
			IntervalMin: interval,
		}

		data, err := json.Marshal(sub)
		if err != nil {
			return err
		}

		return b.Put([]byte(sub.ID), data)
	})
	if err != nil {
		return nil, err
	}

	return &sub, nil
}

func (m *SubscriptionManager) Get(id string) *model.Subscription {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sub *model.Subscription
	_ = m.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return nil
		}

		var s model.Subscription
		if err := json.Unmarshal(data, &s); err == nil {
			sub = &s
		}
		return nil
	})
	return sub
}

func (m *SubscriptionManager) Update(id, name, url string, interval int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("subscription not found: %s", id)
		}

		var sub model.Subscription
		if err := json.Unmarshal(data, &sub); err != nil {
			return err
		}

		if name != "" {
			sub.Name = name
		}
		if url != "" {
			sub.URL = url
		}
		if interval > 0 {
			sub.IntervalMin = interval
		}

		newData, err := json.Marshal(sub)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), newData)
	})
}

func (m *SubscriptionManager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subBucket)
		if b.Get([]byte(id)) == nil {
			return fmt.Errorf("subscription not found: %s", id)
		}
		return b.Delete([]byte(id))
	})
}

func (m *SubscriptionManager) Refresh(id string) error {
	m.mu.Lock()
	var sub *model.Subscription
	_ = m.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return nil
		}
		var s model.Subscription
		if err := json.Unmarshal(data, &s); err == nil {
			sub = &s
		}
		return nil
	})
	m.mu.Unlock()

	if sub == nil {
		return fmt.Errorf("subscription not found: %s", id)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sub.URL, nil)
	if err != nil {
		m.setError(id, err.Error())
		return err
	}

	resp, err := subscriptionHTTPClient.Do(req)
	if err != nil {
		m.setError(id, err.Error())
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		m.setError(id, err.Error())
		return err
	}

	outbounds := parseSubscriptionContent(body)

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("subscription not found: %s", id)
		}

		var s model.Subscription
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}

		s.Outbounds = outbounds
		s.LastUpdated = time.Now()
		s.Error = ""

		newData, err := json.Marshal(s)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), newData)
	})
}

func (m *SubscriptionManager) RefreshAll() []error {
	subs := m.List()
	var errs []error

	for _, sub := range subs {
		if err := m.Refresh(sub.ID); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (m *SubscriptionManager) setError(id, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	_ = m.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return nil
		}

		var s model.Subscription
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}

		s.Error = errMsg

		newData, err := json.Marshal(s)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), newData)
	})
}
