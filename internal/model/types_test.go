package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAuthRequestJSON(t *testing.T) {
	req := AuthRequest{Username: "admin", Password: "pass"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var decoded AuthRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Username != "admin" || decoded.Password != "pass" {
		t.Error("roundtrip failed")
	}
}

func TestAuthResponseJSON(t *testing.T) {
	now := time.Now()
	resp := AuthResponse{Token: "abc", ExpiresAt: now}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var decoded AuthResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Token != "abc" {
		t.Error("token mismatch")
	}
}

func TestServiceStatusJSON(t *testing.T) {
	status := ServiceStatus{Running: true, Uptime: "5m", Memory: 1024, Version: "1.0"}
	data, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ServiceStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.Running || decoded.Uptime != "5m" || decoded.Version != "1.0" {
		t.Error("roundtrip failed")
	}
}

func TestConnectionJSON(t *testing.T) {
	conn := Connection{
		ID: "1", Target: "example.com:443", Outbound: "proxy",
		Upload: 100, Download: 200, Duration: "10s", Rule: "default",
	}
	data, err := json.Marshal(conn)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Connection
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ID != "1" || decoded.Target != "example.com:443" {
		t.Error("roundtrip failed")
	}
}

func TestOutboundJSON(t *testing.T) {
	ob := Outbound{
		Tag: "test", Type: "vless", Server: "1.2.3.4", Port: 443,
		Raw: map[string]any{"uuid": "abc", "tls": map[string]any{"enabled": true}},
	}
	data, err := json.Marshal(ob)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Outbound
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Tag != "test" || decoded.Type != "vless" {
		t.Error("tag/type mismatch")
	}
	rawMap, ok := decoded.Raw.(map[string]any)
	if !ok {
		t.Fatal("Raw is not map[string]any")
	}
	if rawMap["uuid"] != "abc" {
		t.Error("uuid mismatch")
	}
	tlsMap, ok := rawMap["tls"].(map[string]any)
	if !ok {
		t.Fatal("tls is not map[string]any")
	}
	if tlsMap["enabled"] != true {
		t.Error("tls.enabled mismatch")
	}
}

func TestImportResultJSON(t *testing.T) {
	ir := ImportResult{Tag: "vmess-test", Type: "vmess", Server: "1.2.3.4", Port: 443, Config: map[string]any{"uuid": "abc"}}
	data, err := json.Marshal(ir)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ImportResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Tag != "vmess-test" || decoded.Type != "vmess" {
		t.Error("roundtrip failed")
	}
}

func TestTestResultJSON(t *testing.T) {
	tr := TestResult{Tag: "node1", TestType: "tcp", Success: true, LatencyMs: 12.5}
	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatal(err)
	}
	var decoded TestResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.Success || decoded.LatencyMs != 12.5 {
		t.Error("roundtrip failed")
	}
}

func TestSubscriptionJSON(t *testing.T) {
	sub := Subscription{
		ID: "1", Name: "test", URL: "https://example.com/sub",
		IntervalMin: 60, LastUpdated: time.Now(), Error: "",
		Outbounds: []Outbound{{Tag: "node1", Type: "vless", Server: "1.2.3.4", Port: 443}},
	}
	data, err := json.Marshal(sub)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Subscription
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ID != "1" || decoded.Name != "test" {
		t.Error("roundtrip failed")
	}
	if len(decoded.Outbounds) != 1 || decoded.Outbounds[0].Tag != "node1" {
		t.Error("outbounds roundtrip failed")
	}
}
