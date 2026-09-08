package core

import (
	"crypto/aes"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

const importedNilUUID = "00000000-0000-0000-0000-000000000000"

const importedSecretMarker = "private-test-key"

func TestValidateImportedNodePreservesSupportedCredentials(t *testing.T) {
	t.Parallel()
	key := base64.StdEncoding.EncodeToString(make([]byte, aes.BlockSize))
	for _, test := range []struct {
		name string
		kind string
		raw  any
	}{
		{name: "vless empty", kind: "vless"},
		{name: "vless named", kind: "vless", raw: map[string]any{"uuid": "named identity"}},
		{name: "vless vision", kind: "vless", raw: map[string]any{"flow": "xtls-rprx-vision"}},
		{name: "vmess empty", kind: "vmess"},
		{name: "vmess named", kind: "vmess", raw: map[string]any{"uuid": "named identity", "security": "zero"}},
		{name: "tuic nil uuid", kind: "tuic", raw: map[string]any{"uuid": importedNilUUID, "password": ""}},
		{name: "tuic urn uuid", kind: "tuic", raw: map[string]any{"uuid": "urn:uuid:" + importedNilUUID}},
		{name: "ss none", kind: "shadowsocks", raw: map[string]any{"method": "none", "password": ""}},
		{name: "ss password", kind: "shadowsocks", raw: map[string]any{"method": "aes-128-gcm", "password": "example"}},
		{name: "ss key chain", kind: "shadowsocks", raw: map[string]any{"method": "2022-blake3-aes-128-gcm", "password": key + ":" + key}},
		{name: "other protocol empty password", kind: "trojan", raw: map[string]any{"password": ""}},
		{name: "raw type override", kind: "unknown", raw: map[string]any{"type": "vless"}},
		{name: "typed nil raw", kind: "vless", raw: map[string]any(nil)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateImportedNode(importValidationNode(test.kind, test.raw)); err != nil {
				t.Fatalf("supported credentials were rejected: %v", err)
			}
		})
	}
}

func TestValidateImportedNodeRejectsInvalidFields(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		kind string
		raw  any
		path string
	}{
		{name: "vless flow", kind: "vless", raw: map[string]any{"flow": importedSecretMarker}, path: "config.flow"},
		{name: "vmess security", kind: "vmess", raw: map[string]any{"security": importedSecretMarker}, path: "config.security"},
		{name: "tuic missing uuid", kind: "tuic", path: "config.uuid"},
		{name: "tuic invalid uuid", kind: "tuic", raw: map[string]any{"uuid": importedSecretMarker}, path: "config.uuid"},
		{name: "ss missing method", kind: "shadowsocks", path: "config.method"},
		{name: "ss unknown method", kind: "shadowsocks", raw: map[string]any{"method": importedSecretMarker}, path: "config.method"},
		{name: "ss missing password", kind: "shadowsocks", raw: map[string]any{"method": "aes-128-gcm"}, path: "config.password"},
		{name: "ss malformed key", kind: "shadowsocks", path: "config.password",
			raw: map[string]any{"method": "2022-blake3-aes-128-gcm", "password": importedSecretMarker}},
		{name: "ss short key", kind: "shadowsocks", path: "config.password",
			raw: map[string]any{"method": "2022-blake3-aes-128-gcm", "password": "AQ=="}},
		{name: "raw type override", kind: "vless", raw: map[string]any{"type": "tuic"}, path: "config.uuid"},
		{name: "unknown type", kind: "unknown", path: "config"},
		{name: "invalid field type", kind: "vless", raw: map[string]any{"uuid": 7}, path: "config"},
		{name: "raw array", kind: "vless", raw: []string{importedSecretMarker}, path: "config"},
		{name: "raw cannot encode", kind: "vless", raw: map[string]any{"password": func() {}}, path: "config"},
		{name: "raw invalid json", kind: "vless", raw: json.RawMessage(`{"password":"private-test-key"`), path: "config"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateImportedNode(importValidationNode(test.kind, test.raw))
			if err == nil || !strings.HasPrefix(err.Error(), test.path+":") {
				t.Fatalf("want path %s, got %v", test.path, err)
			}
			if strings.Contains(err.Error(), importedSecretMarker) {
				t.Fatal("validation error disclosed credentials")
			}
		})
	}
}

func TestValidateImportedNodePreservesRawInput(t *testing.T) {
	t.Parallel()
	raw := map[string]any{"uuid": "named identity", "server_port": 443}
	before := map[string]any{"uuid": "named identity", "server_port": 443}
	node := importValidationNode("vless", raw)
	node.Port = -1
	if err := ValidateImportedNode(node); err != nil {
		t.Fatalf("raw options must override form values: %v", err)
	}
	if !reflect.DeepEqual(before, raw) {
		t.Fatal("validation modified the caller's raw options")
	}
}

func importValidationNode(kind string, raw any) model.Outbound {
	return model.Outbound{Tag: "node", Type: kind, Server: "192.0.2.1", Port: 443, Raw: raw}
}
