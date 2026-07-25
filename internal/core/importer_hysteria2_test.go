package core

import (
	"context"
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

const (
	testSHA256PinBase64    = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	testSHA256PinBase64Alt = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="
)

func TestParseHysteria2LinkMapsSingBoxOptions(t *testing.T) {
	link := "hysteria2://user:pass@example.com:443/?sni=cdn.example&insecure=1&obfs=salamander&obfs-password=obfs-pass&mport=443-8443&hop_interval=30s&upmbps=50&downmbps=100&pinSHA256=" + url.QueryEscape(testSHA256PinBase64) + "&alpn=h3%2Chq#hy2-full"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	config := result.Config.(map[string]any)
	if config["password"] != "user:pass" {
		t.Fatalf("password = %v", config["password"])
	}
	if ports, ok := config["server_ports"].([]string); !ok || len(ports) != 1 || ports[0] != "443:8443" {
		t.Fatalf("server_ports = %#v", config["server_ports"])
	}
	if config["hop_interval"] != "30s" || config["up_mbps"] != 50 || config["down_mbps"] != 100 {
		t.Fatalf("bandwidth = %#v", config)
	}
	obfs, ok := config["obfs"].(map[string]any)
	if !ok || obfs["type"] != "salamander" || obfs["password"] != "obfs-pass" {
		t.Fatalf("obfs = %#v", config["obfs"])
	}
	tls, ok := config["tls"].(map[string]any)
	if !ok || tls["enabled"] != true || tls["server_name"] != "cdn.example" || tls["insecure"] != true {
		t.Fatalf("tls = %#v", config["tls"])
	}
	if alpn, ok := tls["alpn"].([]string); !ok || len(alpn) != 2 || alpn[0] != "h3" || alpn[1] != "hq" {
		t.Fatalf("alpn = %#v", tls["alpn"])
	}
	if hashes, ok := tls["certificate_public_key_sha256"].([]string); !ok || len(hashes) != 1 || hashes[0] != testSHA256PinBase64 {
		t.Fatalf("pin = %#v", tls["certificate_public_key_sha256"])
	}
}

func TestParseHysteria2OfficialShareLinkPortHopping(t *testing.T) {
	link := "hysteria2://user:pass@better.call:7000-10000,20000/?sni=cdn.example&pinSHA256=" + strings.Repeat("00", 32) + "#official"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if result.Server != "better.call" || result.Port != 7000 {
		t.Fatalf("server = %s:%d", result.Server, result.Port)
	}
	config := result.Config.(map[string]any)
	if !reflect.DeepEqual(config["server_ports"], []string{"7000:10000", "20000:20000"}) {
		t.Fatalf("server ports = %#v", config["server_ports"])
	}
	tls := config["tls"].(map[string]any)
	if !reflect.DeepEqual(tls["certificate_public_key_sha256"], []string{testSHA256PinBase64}) {
		t.Fatalf("certificate pins = %#v", tls["certificate_public_key_sha256"])
	}
}

func TestParseHysteria2LinkProducesSingBoxConfig(t *testing.T) {
	result, err := ParseProxyLink("hysteria2://user:pass@example.com:443?mport=443-8443&pinSHA256=" + url.QueryEscape(testSHA256PinBase64))
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{"outbounds": []any{result.Config}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	defer cancel()
	var config option.Options
	if err := config.UnmarshalJSONContext(ctx, body); err != nil {
		t.Fatalf("generated config rejected by sing-box: %v", err)
	}
}

func TestParseHysteria2LinkUsesAuthQueryAndDefaultsTLS(t *testing.T) {
	result, err := ParseProxyLink("hy2://@example.com:443?auth=query-pass#hy2-auth")
	if err != nil {
		t.Fatal(err)
	}
	config := result.Config.(map[string]any)
	if config["password"] != "query-pass" {
		t.Fatalf("password = %v", config["password"])
	}
	tls := config["tls"].(map[string]any)
	if tls["server_name"] != "example.com" || tls["enabled"] != true {
		t.Fatalf("tls = %#v", tls)
	}
}

func TestParseHysteria2LinkRejectsInvalidOptions(t *testing.T) {
	cases := []string{
		"hysteria2://pass@example.com:443?obfs=quic#bad-obfs",
		"hysteria2://pass@example.com:443?upmbps=fast#bad-bandwidth",
		"hysteria2://pass@example.com:443?network=icmp#bad-network",
		"hysteria2://pass@example.com:443?obfs=salamander#missing-obfs-password",
		"hysteria2://pass@example.com:443?hop_interval=fast#bad-hop-interval",
		"hysteria2://pass@example.com:443?mport=0#bad-port-range",
		"hysteria2://pass@example.com:443?insecure=maybe#bad-insecure",
		"hysteria2://pass@example.com:443?pinSHA256=deadbeef#bad-pin",
	}
	for _, link := range cases {
		_, err := ParseProxyLink(link)
		if err == nil {
			t.Fatalf("link %q unexpectedly succeeded", link)
		}
		if !strings.Contains(err.Error(), "hysteria2") {
			t.Fatalf("link %q error = %v", link, err)
		}
	}
}

func TestHysteria2PasswordAndPortDefaults(t *testing.T) {
	userInfoURL, err := url.Parse("hysteria2://user:pass@example.com")
	if err != nil {
		t.Fatal(err)
	}
	password, err := hysteria2Password(userInfoURL)
	if err != nil || password != "user:pass" {
		t.Fatalf("password = %q, err = %v", password, err)
	}

	queryURL, err := url.Parse("hy2://@example.com?auth=query%3Apass")
	if err != nil {
		t.Fatal(err)
	}
	password, err = hysteria2Password(queryURL)
	if err != nil || password != "query:pass" {
		t.Fatalf("query password = %q, err = %v", password, err)
	}
	passwordURL, err := url.Parse("hy2://@example.com?password=query-password")
	if err != nil {
		t.Fatal(err)
	}
	password, err = hysteria2Password(passwordURL)
	if err != nil || password != "query-password" {
		t.Fatalf("password alias = %q, err = %v", password, err)
	}
	if port, err := parseHysteria2Port(queryURL); err != nil || port != 443 {
		t.Fatalf("default port = %d, err = %v", port, err)
	}

	portURL := &url.URL{Host: "example.com:8443"}
	if port, err := parseHysteria2Port(portURL); err != nil || port != 8443 {
		t.Fatalf("explicit port = %d, err = %v", port, err)
	}
	invalidURL := &url.URL{Host: "example.com:70000"}
	if _, err := parseHysteria2Port(invalidURL); err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestNormalizeHysteria2PortRange(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "443-8443", want: "443:8443"},
		{raw: "443:8443", want: "443:8443"},
		{raw: "443", want: "443:443"},
		{raw: ":1024", want: ":1024"},
		{raw: "1024:", want: "1024:"},
	}
	for _, test := range tests {
		got, err := normalizeHysteria2PortRange(test.raw)
		if err != nil || got != test.want {
			t.Errorf("normalizeHysteria2PortRange(%q) = %q, %v; want %q", test.raw, got, err, test.want)
		}
	}
	for _, raw := range []string{"", ":", "-1", "1-", "1--2", "8443-443", "0:1", "1:65536", "1:2:3", "x:2", "1-x"} {
		if _, err := normalizeHysteria2PortRange(raw); err == nil {
			t.Errorf("normalizeHysteria2PortRange(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestNormalizeHysteria2Duration(t *testing.T) {
	tests := map[string]string{"30s": "30s", "30": "30s", " 1m ": "1m"}
	for raw, want := range tests {
		got, err := normalizeHysteria2Duration(raw)
		if err != nil || got != want {
			t.Errorf("normalizeHysteria2Duration(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
	for _, raw := range []string{"", "fast", "-1s"} {
		if _, err := normalizeHysteria2Duration(raw); err == nil {
			t.Errorf("normalizeHysteria2Duration(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestHysteria2TLSAndOptionAliases(t *testing.T) {
	query := url.Values{
		"server_name":      {"sni.example"},
		"skip_cert_verify": {"yes"},
		"pin_sha256":       {testSHA256PinBase64 + "," + testSHA256PinBase64Alt},
		"up_mbps":          {"12"},
		"down-mbps":        {"34"},
		"server_ports":     {"1000-1002,2000:2001"},
		"network":          {"TCP,udp,tcp"},
	}
	options, err := buildHysteria2Options(query, "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if options["up_mbps"] != 12 || options["down_mbps"] != 34 {
		t.Fatalf("bandwidth options = %#v", options)
	}
	if !reflect.DeepEqual(options["server_ports"], []string{"1000:1002", "2000:2001"}) {
		t.Fatalf("server ports = %#v", options["server_ports"])
	}
	if !reflect.DeepEqual(options["network"], []string{"tcp", "udp"}) {
		t.Fatalf("network = %#v", options["network"])
	}
	tls, ok := options["tls"].(map[string]any)
	if !ok || tls["server_name"] != "sni.example" || tls["insecure"] != true {
		t.Fatalf("tls = %#v", options["tls"])
	}
	if !reflect.DeepEqual(tls["certificate_public_key_sha256"], []string{testSHA256PinBase64, testSHA256PinBase64Alt}) {
		t.Fatalf("pins = %#v", tls["certificate_public_key_sha256"])
	}
}

func TestHysteria2ObfsAndNetworkValidation(t *testing.T) {
	obfs, err := parseHysteria2Obfs(url.Values{"obfs": {"SALAMANDER"}, "obfs_password": {"secret"}})
	if err != nil || obfs["type"] != "salamander" || obfs["password"] != "secret" {
		t.Fatalf("obfs = %#v, err = %v", obfs, err)
	}
	if obfs, err := parseHysteria2Obfs(url.Values{}); err != nil || obfs != nil {
		t.Fatalf("empty obfs = %#v, err = %v", obfs, err)
	}
	for _, query := range []url.Values{
		{"obfs-password": {"secret"}},
		{"obfs": {"salamander"}},
		{"obfs": {"quic"}},
	} {
		if _, err := parseHysteria2Obfs(query); err == nil {
			t.Errorf("parseHysteria2Obfs(%v) unexpectedly succeeded", query)
		}
	}
	if network, err := parseHysteria2Network(url.Values{"network": {"udp"}}); err != nil || network != "udp" {
		t.Fatalf("single network = %#v, err = %v", network, err)
	}
	if network, err := parseHysteria2Network(url.Values{}); err != nil || network != nil {
		t.Fatalf("empty network = %#v, err = %v", network, err)
	}
	for _, raw := range []string{"  ", "icmp", "tcp,icmp"} {
		if _, err := parseHysteria2Network(url.Values{"network": {raw}}); err == nil {
			t.Errorf("network %q unexpectedly succeeded", raw)
		}
	}
}

func TestParseHysteria2BoolAndMbps(t *testing.T) {
	for _, raw := range []string{"1", "true", "YES", "on"} {
		value, err := parseHysteria2Bool(url.Values{"insecure": {raw}}, "insecure")
		if err != nil || !value {
			t.Errorf("true value %q = %v, %v", raw, value, err)
		}
	}
	for _, raw := range []string{"0", "false", "NO", "off"} {
		value, err := parseHysteria2Bool(url.Values{"insecure": {raw}}, "insecure")
		if err != nil || value {
			t.Errorf("false value %q = %v, %v", raw, value, err)
		}
	}
	if value, err := parseHysteria2Bool(url.Values{}, "insecure"); err != nil || value {
		t.Fatalf("missing bool = %v, %v", value, err)
	}
	if _, err := parseHysteria2Bool(url.Values{"insecure": {"maybe"}}, "insecure"); err == nil {
		t.Fatal("expected invalid bool error")
	}
	if value, present, err := parseHysteria2Mbps(url.Values{}, "upmbps"); err != nil || present || value != 0 {
		t.Fatalf("missing Mbps = %d, %v, %v", value, present, err)
	}
	if value, present, err := parseHysteria2Mbps(url.Values{"upmbps": {"0"}}, "upmbps"); err != nil || !present || value != 0 {
		t.Fatalf("zero Mbps = %d, %v, %v", value, present, err)
	}
	for _, raw := range []string{"fast", "-1"} {
		if _, _, err := parseHysteria2Mbps(url.Values{"upmbps": {raw}}, "upmbps"); err == nil {
			t.Errorf("Mbps %q unexpectedly succeeded", raw)
		}
	}
}
