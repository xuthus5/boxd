package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

const maxSubscriptionBodyBytes = 16 << 20

type subscriptionRefreshData struct {
	outbounds []model.Outbound
	traffic   *model.SubscriptionTraffic
}

func (m *SubscriptionManager) Refresh(id string) error {
	return m.RefreshContext(context.Background(), id)
}

// RefreshContext refreshes one subscription and honors caller cancellation.
func (m *SubscriptionManager) RefreshContext(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sub := m.Get(id)
	if sub == nil {
		return newSubscriptionRefreshError(SubRefreshNotFound, fmt.Sprintf("subscription not found: %s", id), 0)
	}
	outbounds, traffic, err := downloadSubscriptionOutbounds(ctx, m.client, sub.URL)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		classified := classifySubscriptionRefreshError(err)
		return m.recordRefreshErrorIfUnchanged(id, sub, classified)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(outbounds) == 0 {
		classified := newSubscriptionRefreshError(SubRefreshEmpty, "subscription content produced no nodes", 0)
		return m.recordRefreshErrorIfUnchanged(id, sub, classified)
	}
	return m.saveRefreshedSubscriptionIfUnchanged(id, sub, subscriptionRefreshData{
		outbounds: outbounds,
		traffic:   traffic,
	})
}

func (m *SubscriptionManager) recordRefreshError(id string, refreshErr *SubscriptionRefreshError) error {
	return m.recordRefreshErrorIfUnchanged(id, nil, refreshErr)
}

func (m *SubscriptionManager) recordRefreshErrorIfUnchanged(
	id string,
	expected *model.Subscription,
	refreshErr *SubscriptionRefreshError,
) error {
	applied, err := m.setErrorIfUnchanged(id, expected, refreshErr)
	if err != nil {
		return errors.Join(refreshErr, fmt.Errorf("persisting subscription refresh error: %w", err))
	}
	if !applied {
		return nil
	}
	return refreshErr
}

func downloadSubscriptionOutbounds(
	parent context.Context,
	client *http.Client,
	rawURL string,
) ([]model.Outbound, *model.SubscriptionTraffic, error) {
	if err := ValidateSubscriptionURL(rawURL); err != nil {
		return nil, nil, err
	}
	if client == nil {
		client = newSubscriptionHTTPClient()
	}
	ctx, cancel := context.WithTimeout(parent, subscriptionHTTPTimeout)
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

func (m *SubscriptionManager) saveRefreshedSubscriptionIfUnchanged(
	id string,
	expected *model.Subscription,
	refreshData subscriptionRefreshData,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subBucket)
		subscription, err := decodeStoredSubscription(b, id)
		if err != nil {
			return err
		}
		if subscription == nil {
			return fmt.Errorf("subscription not found: %s", id)
		}
		if !subscriptionMatchesSnapshot(subscription, expected) {
			return nil
		}
		subscription.Outbounds = refreshData.outbounds
		subscription.Traffic = refreshData.traffic
		subscription.LastUpdated = time.Now()
		subscription.Error = ""
		subscription.ErrorCode = ""
		subscription.ErrorAt = nil
		newData, err := json.Marshal(subscription)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), newData)
	})
}

func (m *SubscriptionManager) RefreshAll() []SubscriptionRefreshFailure {
	return m.RefreshAllContext(context.Background())
}

// RefreshAllContext refreshes subscriptions with bounded concurrency until completion or cancellation.
func (m *SubscriptionManager) RefreshAllContext(ctx context.Context) []SubscriptionRefreshFailure {
	subs, err := m.List()
	if err != nil {
		return []SubscriptionRefreshFailure{{
			Code:    SubRefreshUnknown,
			Message: err.Error(),
		}}
	}
	if ctx.Err() != nil || len(subs) == 0 {
		return nil
	}
	return m.refreshSubscriptionsConcurrently(ctx, subs).failures
}

func (m *SubscriptionManager) setError(id string, refreshErr *SubscriptionRefreshError) error {
	_, err := m.setErrorIfUnchanged(id, nil, refreshErr)
	return err
}

func (m *SubscriptionManager) setErrorIfUnchanged(
	id string,
	expected *model.Subscription,
	refreshErr *SubscriptionRefreshError,
) (bool, error) {
	if refreshErr == nil {
		return false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	applied := false
	err := m.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subBucket)
		subscription, err := decodeStoredSubscription(b, id)
		if err != nil {
			return err
		}
		if subscription == nil || !subscriptionMatchesSnapshot(subscription, expected) {
			return nil
		}

		now := time.Now().UTC()
		subscription.Error = refreshErr.Message
		subscription.ErrorCode = refreshErr.Code
		subscription.ErrorAt = &now

		newData, err := json.Marshal(subscription)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(id), newData); err != nil {
			return err
		}
		applied = true
		return nil
	})
	return applied, err
}

func decodeStoredSubscription(bucket *bbolt.Bucket, id string) (*model.Subscription, error) {
	if bucket == nil {
		return nil, errors.New("subscriptions bucket is missing")
	}
	data := bucket.Get([]byte(id))
	if data == nil {
		return nil, nil
	}
	var subscription model.Subscription
	if err := json.Unmarshal(data, &subscription); err != nil {
		return nil, err
	}
	return &subscription, nil
}

func subscriptionMatchesSnapshot(current, expected *model.Subscription) bool {
	return expected == nil || reflect.DeepEqual(current, expected)
}
