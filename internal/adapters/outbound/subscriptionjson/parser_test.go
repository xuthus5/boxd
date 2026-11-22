package subscriptionjson_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/adapters/outbound/subscriptionjson"
)

const sampleSubscription = `{
  "outbounds": [
    {"type":"selector","tag":"auto","outbounds":["hk"]},
    {"type":"shadowsocks","tag":"Hong Kong","server":"1.2.3.4","server_port":8388,"method":"aes-128-gcm","password":"secret"},
    {"type":"trojan","tag":"Japan","server":"example.com","server_port":"443","password":"secret","network":"tcp"}
  ]
}`

func TestParseSingBoxJSONAndBase64(t *testing.T) {
	parser := subscriptionjson.New()
	for name, body := range map[string][]byte{
		"json":   []byte(sampleSubscription),
		"base64": []byte(base64.StdEncoding.EncodeToString([]byte(sampleSubscription))),
	} {
		t.Run(name, func(t *testing.T) {
			nodes, err := parser.Parse(context.Background(), control.DownloadResult{Body: body})
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(nodes) != 2 {
				t.Fatalf("nodes = %+v, want two proxy outbounds", nodes)
			}
			if nodes[0].Name != "Hong Kong" || nodes[0].Protocol != "shadowsocks" || nodes[0].Server != "1.2.3.4" ||
				nodes[0].Port != 8388 || nodes[0].Options["method"] != "aes-128-gcm" {
				t.Fatalf("first node = %+v", nodes[0])
			}
			if _, exists := nodes[0].Options["type"]; exists {
				t.Fatalf("reserved option leaked into node: %+v", nodes[0].Options)
			}
			if nodes[1].Port != 443 || nodes[1].NodeSupportsUDP() {
				t.Fatalf("second node = %+v", nodes[1])
			}
		})
	}
}

func TestParseRejectsInvalidDocuments(t *testing.T) {
	parser := subscriptionjson.New()
	if _, err := parser.Parse(context.Background(), control.DownloadResult{Body: []byte("not a subscription")}); err == nil {
		t.Fatal("Parse() accepted invalid content")
	}
	if _, err := parser.Parse(context.Background(), control.DownloadResult{Body: []byte(`{"outbounds":[{"type":"trojan","server":"example.com","server_port":70000}]}`)}); err == nil {
		t.Fatal("Parse() accepted an invalid port")
	}
}
