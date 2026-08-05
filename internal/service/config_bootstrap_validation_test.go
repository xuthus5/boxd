package service

import (
	"context"
	"testing"
)

// TestBootstrapConfigPassesValidation 验证 boxd 自动生成的默认配置能通过内核校验。
func TestBootstrapConfigPassesValidation(t *testing.T) {
	body := []byte(`{
  "log": {"level": "info", "timestamp": true},
  "inbounds": [{"type": "mixed", "tag": "mixed-in", "listen": "::", "listen_port": 1080}],
  "outbounds": [
    {"type": "direct", "tag": "direct"},
    {"type": "block", "tag": "block"}
  ],
  "route": {"final": "direct"}
}`)
	if err := ValidateRuntimeConfig(context.Background(), body); err != nil {
		t.Fatalf("bootstrap config failed validation: %v", err)
	}
}
