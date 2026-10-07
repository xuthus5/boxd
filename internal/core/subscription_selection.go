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
