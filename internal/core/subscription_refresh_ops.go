package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

const maxSubscriptionBodyBytes = 16 << 20

func (m *SubscriptionManager) Refresh(id string) error {
	sub := m.Get(id)
	if sub == nil {
		return newSubscriptionRefreshError(SubRefreshNotFound, fmt.Sprintf("subscription not found: %s", id), 0)
	}
	outbounds, traffic, err := downloadSubscriptionOutbounds(m.client, sub.URL)
	if err != nil {
		classified := classifySubscriptionRefreshError(err)
		return m.recordRefreshError(id, classified)
	}
	if len(outbounds) == 0 {
		classified := newSubscriptionRefreshError(SubRefreshEmpty, "subscription content produced no nodes", 0)
		return m.recordRefreshError(id, classified)
	}
	return m.saveRefreshedSubscription(id, outbounds, traffic)
}

func (m *SubscriptionManager) recordRefreshError(id string, refreshErr *SubscriptionRefreshError) error {
	if err := m.setError(id, refreshErr); err != nil {
		return errors.Join(refreshErr, fmt.Errorf("persisting subscription refresh error: %w", err))
	}
	return refreshErr
}

func downloadSubscriptionOutbounds(client *http.Client, rawURL string) ([]model.Outbound, *model.SubscriptionTraffic, error) {
	if err := ValidateSubscriptionURL(rawURL); err != nil {
		return nil, nil, err
	}
	if client == nil {
		client = newSubscriptionHTTPClient()
	}
	ctx, cancel := context.WithTimeout(context.Background(), subscriptionHTTPTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, nil, classifySubscriptionRefreshError(err)
	}
	// 部分机场根据 UA 返回不同内容；使用常见 clash 兼容 UA 提高兼容性。
	req.Header.Set("User-Agent", "clash.meta")
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, classifySubscriptionRefreshError(err)
	}
	if resp == nil || resp.Body == nil {
		return nil, nil, errors.New("subscription response body is nil")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, classifyHTTPStatus(resp.StatusCode)
	}
	if resp.ContentLength > maxSubscriptionBodyBytes {
		return nil, nil, newSubscriptionRefreshError(SubRefreshContentTooLarge, "subscription content is too large", 0)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSubscriptionBodyBytes+1))
	if err != nil {
		return nil, nil, classifySubscriptionRefreshError(err)
	}
	if int64(len(body)) > maxSubscriptionBodyBytes {
		return nil, nil, newSubscriptionRefreshError(SubRefreshContentTooLarge, "subscription content is too large", 0)
	}
	return parseSubscriptionContent(body), parseSubscriptionUserinfo(resp.Header), nil
}

func (m *SubscriptionManager) saveRefreshedSubscription(
	id string,
	outbounds []model.Outbound,
	traffic *model.SubscriptionTraffic,
) error {
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
		s.Traffic = traffic
		s.LastUpdated = time.Now()
		s.Error = ""
		s.ErrorCode = ""
		s.ErrorAt = nil
		newData, err := json.Marshal(s)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), newData)
	})
}

func (m *SubscriptionManager) RefreshAll() []SubscriptionRefreshFailure {
	subs, err := m.List()
	if err != nil {
		return []SubscriptionRefreshFailure{{
			Code:    SubRefreshUnknown,
			Message: err.Error(),
		}}
	}
	var failures []SubscriptionRefreshFailure
	for _, sub := range subs {
		if err := m.Refresh(sub.ID); err != nil {
			classified := classifySubscriptionRefreshError(err)
			failures = append(failures, SubscriptionRefreshFailure{
				ID:      sub.ID,
				Name:    sub.Name,
				Code:    classified.Code,
				Message: err.Error(),
			})
		}
	}
	return failures
}

func (m *SubscriptionManager) setError(id string, refreshErr *SubscriptionRefreshError) error {
	if refreshErr == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return nil
		}

		var s model.Subscription
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}

		now := time.Now().UTC()
		s.Error = refreshErr.Message
		s.ErrorCode = refreshErr.Code
		s.ErrorAt = &now

		newData, err := json.Marshal(s)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), newData)
	})
}
