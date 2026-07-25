package core

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestParseShadowsocksSIP002Credentials(t *testing.T) {
	result, err := ParseProxyLink("ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@192.168.1.1:8080#ss-test")
	if err != nil {
		t.Fatal(err)
	}
	config := result.Config.(map[string]any)
	if config["method"] != "aes-256-gcm" || config["password"] != "password" {
		t.Fatalf("credentials = %#v", config)
	}
}

func TestParseShadowsocksPlainCredentialsAndPlugin(t *testing.T) {
	result, err := ParseProxyLink("ss://aes-256-gcm:secret%20pass@server.example.com:8388?plugin=obfs-local&plugin_opts=obfs%3Dhttp#plain")
	if err != nil {
		t.Fatal(err)
	}
	config := result.Config.(map[string]any)
	if config["method"] != "aes-256-gcm" || config["password"] != "secret pass" {
		t.Fatalf("credentials = %#v", config)
	}
	if config["plugin"] != "obfs-local" || config["plugin_opts"] != "obfs=http" {
		t.Fatalf("plugin = %#v", config)
	}
}

func TestParseShadowsocksLegacyBase64Link(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte("chacha20-ietf-poly1305:legacy-pass@example.com:443"))
	result, err := ParseProxyLink("ss://" + payload + "#legacy")
	if err != nil {
		t.Fatal(err)
	}
	config := result.Config.(map[string]any)
	if result.Server != "example.com" || result.Port != 443 {
		t.Fatalf("server = %s:%d", result.Server, result.Port)
	}
	if config["method"] != "chacha20-ietf-poly1305" || config["password"] != "legacy-pass" {
		t.Fatalf("credentials = %#v", config)
	}
}

func TestParseShadowsocksRejectsInvalidCredentials(t *testing.T) {
	for _, link := range []string{
		"ss://not-base64@server.example.com:8388#invalid",
		"ss://YWVzLTI1Ni1nY20=@server.example.com:8388#missing-password",
		"ss://server.example.com:8388#missing-credentials",
	} {
		_, err := ParseProxyLink(link)
		if err == nil || !strings.Contains(err.Error(), "credentials") {
			t.Fatalf("link %q error = %v", link, err)
		}
	}
}
