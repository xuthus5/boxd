package singbox

import (
	"context"
	"testing"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/adapters/outbound/singboxconfig"
)

func TestGeneratedManagedConfigParsesWithSingBox(t *testing.T) {
	state := control.DefaultDesiredState()
	state.Nodes = []control.Node{
		{
			ID:       "hk-01",
			Name:     "Hong Kong",
			Protocol: "shadowsocks",
			Server:   "127.0.0.1",
			Port:     8388,
			Options: map[string]any{
				"method":   "aes-128-gcm",
				"password": "secret",
			},
		},
		{
			ID:       "jp-02",
			Name:     "Japan",
			Protocol: "shadowsocks",
			Server:   "127.0.0.2",
			Port:     8389,
			Options: map[string]any{
				"method":   "aes-128-gcm",
				"password": "secret",
			},
		},
	}
	state.Groups = []control.Group{{
		ID:      "smart",
		Name:    "Smart",
		Kind:    control.GroupKindSelector,
		Members: []string{"hk-01", "jp-02"},
	}}
	state.Chains = []control.ProxyChain{{
		ID:   "relay",
		Name: "Relay",
		Hops: []control.ProxyHop{
			{Kind: control.EgressKindNode, ID: "hk-01"},
			{Kind: control.EgressKindGroup, ID: "smart"},
		},
	}}
	state.Routes.DefaultEgress = control.EgressRef{Kind: control.EgressKindChain, ID: "relay"}
	state.Routes.Presets.AdBlock = true
	state.Routes.Rules = []control.RouteRule{{
		ID:     "media",
		Name:   "Media",
		Action: control.RouteActionEgress,
		Egress: control.EgressRef{Kind: control.EgressKindGroup, ID: "smart"},
		Match: control.RouteMatch{
			DomainSuffixes: []string{".example.com"},
			ProcessNames:   []string{"chrome.exe"},
			Ports:          []uint16{80, 443},
		},
	}}

	document, err := singboxconfig.New().Compile(context.Background(), state)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if _, err := parseOptions(context.Background(), document, t.TempDir()); err != nil {
		t.Fatalf("generated config does not parse with sing-box v1.14.4: %v\n%s", err, document.Body)
	}
}
