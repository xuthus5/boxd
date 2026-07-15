package api

import (
	"encoding/json"
	"os"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

func setSubscriptionOutbounds(
	t *testing.T,
	manager *core.SubscriptionManager,
	id string,
	outbounds []model.Outbound,
) {
	t.Helper()
	subscription := manager.Get(id)
	if subscription == nil {
		t.Fatalf("subscription %q not found", id)
	}
	subscription.Outbounds = outbounds
	data, err := json.Marshal(subscription)
	if err != nil {
		t.Fatalf("encoding subscription: %v", err)
	}
	if err := manager.DB().Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte("subscriptions")).Put([]byte(id), data)
	}); err != nil {
		t.Fatalf("saving subscription: %v", err)
	}
}

func readConfigMap(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	config := map[string]any{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("decoding config: %v", err)
	}
	return config
}

func outboundByTag(t *testing.T, config map[string]any, tag string) map[string]any {
	t.Helper()
	outbound := optionalOutboundByTag(config, tag)
	if outbound == nil {
		t.Fatalf("outbound %q not found in %#v", tag, config["outbounds"])
	}
	return outbound
}

func optionalOutboundByTag(config map[string]any, tag string) map[string]any {
	outbounds, ok := config["outbounds"].([]any)
	if !ok {
		return nil
	}
	for _, entry := range outbounds {
		outbound, ok := entry.(map[string]any)
		if ok && outbound["tag"] == tag {
			return outbound
		}
	}
	return nil
}
