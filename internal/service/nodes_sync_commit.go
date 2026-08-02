package service

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/xuthus5/boxd/internal/core"
)

func readSyncConfig(configPath string) (map[string]any, []byte, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, nil, err
	}
	config := map[string]any{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, nil, err
	}
	return config, data, nil
}

type managedGroupStore interface {
	SetURLTestManagedGroups([]string) error
}

type syncCommit struct {
	path     string
	previous []byte
	groups   managedGroupStore
}

func (c syncCommit) write(config map[string]any, groupTags []string) error {
	written, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := atomicWriteFile(c.path, written); err != nil {
		return err
	}
	if err := c.groups.SetURLTestManagedGroups(groupTags); err != nil {
		return errors.Join(err, atomicWriteFile(c.path, c.previous))
	}
	return nil
}

func isProxyLikeOutboundType(typ string) bool {
	switch typ {
	case "vless", "vmess", "trojan", "shadowsocks", "hysteria", "hysteria2", "tuic", "shadowtls", "anytls", "ssh", "tor":
		return true
	default:
		return false
	}
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// SyncOutboundsAndRestart 重建托管出站配置并在需要时重启内核，失败时回滚。
func SyncOutboundsAndRestart(
	nodeManager *core.NodeManager,
	subManager *core.SubscriptionManager,
	configPath string,
	instance restartable,
) error {
	return syncOutboundsAndRestart(nodeManager, subManager, configPath, instance)
}
