package core

import (
	"encoding/json"
	"fmt"
	"slices"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

// IsProxyLikeOutboundType 判断出站类型是否属于可作为订阅出口的代理协议。
func IsProxyLikeOutboundType(typ string) bool {
	switch typ {
	case "snell", "vless", "vmess", "trojan", "shadowsocks", "hysteria", "hysteria2", "tuic", "shadowtls", "anytls", "ssh", "tor":
		return true
	default:
		return false
	}
}

// SubscriptionSelectableTags 返回订阅中可作为出口选择的节点 tag，保持节点原始顺序。
func SubscriptionSelectableTags(subscription model.Subscription) []string {
	tags := make([]string, 0, len(subscription.Outbounds))
	for _, outbound := range subscription.Outbounds {
		if outbound.Tag != "" && IsProxyLikeOutboundType(outbound.Type) {
			tags = append(tags, outbound.Tag)
		}
	}
	return tags
}

// ValidateSubscriptionSelection 校验所选出口是否属于订阅节点；空 tag 表示清除选择。
func ValidateSubscriptionSelection(subscription model.Subscription, tag string) error {
	if tag == "" {
		return nil
	}
	if !slices.Contains(SubscriptionSelectableTags(subscription), tag) {
		return fmt.Errorf("outbound %q is not part of subscription %q", tag, subscription.Name)
	}
	return nil
}

// ResolveSubscriptionSelectorDefault 返回 selector 组可用的默认出口，节点不可用时返回空串。
func ResolveSubscriptionSelectorDefault(members []string, selected string) string {
	if selected != "" && slices.Contains(members, selected) {
		return selected
	}
	return ""
}

// ResolveSubscriptionSelectorTarget returns the runtime selector target. An empty or stale persisted
// selection falls back to the first member, matching sing-box's selector behaviour without a default.
func ResolveSubscriptionSelectorTarget(members []string, selected string) string {
	if target := ResolveSubscriptionSelectorDefault(members, selected); target != "" {
		return target
	}
	if len(members) > 0 {
		return members[0]
	}
	return ""
}

// ManagedSubscriptionSelectorTargets returns the authoritative runtime selector target for every
// managed subscription selector. URLTest groups and subscriptions without proxy members are skipped.
func ManagedSubscriptionSelectorTargets(manager *SubscriptionManager) (map[string]string, error) {
	if manager == nil {
		return map[string]string{}, nil
	}
	subscriptions, err := manager.List()
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defaults, err := NewSettingsManager(manager.DB()).URLTestDefaults()
	if err != nil {
		return nil, err
	}
	targets := make(map[string]string, len(subscriptions))
	for _, subscription := range subscriptions {
		if subscription.Name == "" || ResolveURLTest(defaults, subscription.URLTest).Enabled {
			continue
		}
		target := ResolveSubscriptionSelectorTarget(SubscriptionSelectableTags(subscription), subscription.Selected)
		if target != "" {
			targets[subscription.Name] = target
		}
	}
	return targets, nil
}

// SetSelected 持久化订阅在 selector 模式下选中的出口节点，空 tag 表示清除选择。
func (m *SubscriptionManager) SetSelected(id, tag string) (*model.Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var updated model.Subscription
	err := m.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(subBucket)
		subscription, err := decodeStoredSubscription(bucket, id)
		if err != nil {
			return err
		}
		if subscription == nil {
			return fmt.Errorf("subscription not found: %s", id)
		}
		subscription.Selected = tag
		data, err := json.Marshal(subscription)
		if err != nil {
			return err
		}
		if err := bucket.Put([]byte(id), data); err != nil {
			return err
		}
		updated = *subscription
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}
