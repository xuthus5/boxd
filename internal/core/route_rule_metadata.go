package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

const (
	MaxRouteRuleNameLength        = 100
	MaxRouteRuleDescriptionLength = 500
)

var (
	ErrInvalidRouteRuleMetadata = errors.New("invalid route rule metadata")
	routeRuleMetadataBucket     = []byte("route_rule_metadata")
	routeRuleMetadataKey        = []byte("rules")
	routeRuleMetadataInitKey    = []byte("default_names_initialized")
)

type storedRouteRuleMetadata struct {
	Fingerprint string `json:"fingerprint"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RouteRuleMetadataManager struct {
	db *bbolt.DB
}

func NewRouteRuleMetadataManager(db *bbolt.DB) *RouteRuleMetadataManager {
	return &RouteRuleMetadataManager{db: db}
}

func routeRuleFingerprint(rule any) (string, error) {
	data, err := json.Marshal(rule)
	if err != nil {
		return "", fmt.Errorf("encoding route rule: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (m *RouteRuleMetadataManager) stored() ([]storedRouteRuleMetadata, error) {
	var entries []storedRouteRuleMetadata
	err := m.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(routeRuleMetadataBucket)
		if bucket == nil {
			return nil
		}
		data := bucket.Get(routeRuleMetadataKey)
		if data == nil {
			return nil
		}
		if err := json.Unmarshal(data, &entries); err != nil {
			return fmt.Errorf("decoding route rule metadata: %w", err)
		}
		return nil
	})
	return entries, err
}

func (m *RouteRuleMetadataManager) List(rules []any) ([]model.RouteRuleMetadata, error) {
	stored, err := m.stored()
	if err != nil {
		return nil, err
	}
	result := make([]model.RouteRuleMetadata, len(rules))
	matched := make([]bool, len(rules))
	used := make([]bool, len(stored))
	for index, rule := range rules {
		fingerprint, fingerprintErr := routeRuleFingerprint(rule)
		if fingerprintErr != nil {
			return nil, fingerprintErr
		}
		for storedIndex, entry := range stored {
			if used[storedIndex] || entry.Fingerprint != fingerprint {
				continue
			}
			result[index] = publicRouteRuleMetadata(entry)
			matched[index] = true
			used[storedIndex] = true
			break
		}
	}
	for index := range rules {
		if matched[index] || index >= len(stored) || used[index] {
			continue
		}
		result[index] = publicRouteRuleMetadata(stored[index])
		used[index] = true
	}
	return result, nil
}

func publicRouteRuleMetadata(entry storedRouteRuleMetadata) model.RouteRuleMetadata {
	return model.RouteRuleMetadata{Name: entry.Name, Description: entry.Description}
}

func validateRouteRuleMetadata(metadata model.RouteRuleMetadata) (model.RouteRuleMetadata, error) {
	metadata.Name = strings.TrimSpace(metadata.Name)
	metadata.Description = strings.TrimSpace(metadata.Description)
	if len([]rune(metadata.Name)) > MaxRouteRuleNameLength {
		return model.RouteRuleMetadata{}, fmt.Errorf("%w: route rule name is too long", ErrInvalidRouteRuleMetadata)
	}
	if len([]rune(metadata.Description)) > MaxRouteRuleDescriptionLength {
		return model.RouteRuleMetadata{}, fmt.Errorf("%w: route rule description is too long", ErrInvalidRouteRuleMetadata)
	}
	return metadata, nil
}

func (m *RouteRuleMetadataManager) Save(rules []any, metadata []model.RouteRuleMetadata) error {
	if len(rules) != len(metadata) {
		return fmt.Errorf("%w: metadata count does not match rules", ErrInvalidRouteRuleMetadata)
	}
	stored := make([]storedRouteRuleMetadata, len(rules))
	for index, rule := range rules {
		validated, err := validateRouteRuleMetadata(metadata[index])
		if err != nil {
			return err
		}
		fingerprint, err := routeRuleFingerprint(rule)
		if err != nil {
			return err
		}
		stored[index] = storedRouteRuleMetadata{Fingerprint: fingerprint, Name: validated.Name, Description: validated.Description}
	}
	data, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("encoding route rule metadata: %w", err)
	}
	if err := m.db.Update(func(tx *bbolt.Tx) error {
		bucket, createErr := tx.CreateBucketIfNotExists(routeRuleMetadataBucket)
		if createErr != nil {
			return createErr
		}
		return bucket.Put(routeRuleMetadataKey, data)
	}); err != nil {
		return fmt.Errorf("saving route rule metadata: %w", err)
	}
	return nil
}

func (m *RouteRuleMetadataManager) ApplyDefaultNames(rules []any) error {
	metadata, err := m.List(rules)
	if err != nil {
		return err
	}
	for index, rule := range rules {
		if metadata[index].Name != "" {
			continue
		}
		if object, ok := rule.(map[string]any); ok {
			metadata[index].Name = defaultRouteRuleName(object)
		}
	}
	return m.Save(rules, metadata)
}

func (m *RouteRuleMetadataManager) InitializeDefaultNames(rules []any) error {
	initialized := false
	if err := m.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(routeRuleMetadataBucket)
		initialized = bucket != nil && bucket.Get(routeRuleMetadataInitKey) != nil
		return nil
	}); err != nil {
		return fmt.Errorf("checking route rule metadata initialization: %w", err)
	}
	if initialized {
		return nil
	}
	if err := m.ApplyDefaultNames(rules); err != nil {
		return err
	}
	if err := m.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(routeRuleMetadataBucket)
		return bucket.Put(routeRuleMetadataInitKey, []byte("true"))
	}); err != nil {
		return fmt.Errorf("marking route rule metadata initialized: %w", err)
	}
	return nil
}

func defaultRouteRuleName(rule map[string]any) string {
	action, _ := rule["action"].(string)
	if action == "sniff" {
		return "协议嗅探"
	}
	if action == "hijack-dns" {
		return "DNS 请求劫持"
	}
	if action == "resolve" {
		return "域名解析后分流"
	}
	switch strings.ToLower(stringValue(rule["clash_mode"])) {
	case "direct":
		return "直连模式"
	case "global":
		return "全局代理模式"
	}
	if private, _ := rule["ip_is_private"].(bool); private {
		return "私有地址直连"
	}
	network, _ := rule["network"].(string)
	if network == "icmp" {
		return "ICMP 流量直连"
	}
	if network == "udp" && portValue(rule["port"]) == "443" && action == "reject" {
		return "QUIC 流量拦截"
	}
	switch ruleSetTag(rule) {
	case "loyalsoldier-reject", "geosite-category-ads-all":
		return "广告流量拦截"
	case "loyalsoldier-direct", "geosite-cn":
		return "中国域名直连"
	case "geoip-cn":
		return "中国 IP 直连"
	case "loyalsoldier-proxy":
		return "代理列表流量走代理"
	case "geosite-google-play":
		return "Google Play 流量走代理"
	default:
		return ""
	}
}

func ruleSetTag(rule map[string]any) string {
	switch values := rule["rule_set"].(type) {
	case []string:
		if len(values) > 0 {
			return values[0]
		}
	case []any:
		if len(values) > 0 {
			tag, _ := values[0].(string)
			return tag
		}
	}
	return ""
}
