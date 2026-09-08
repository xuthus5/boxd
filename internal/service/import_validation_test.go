package service

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

type importValidationCase struct {
	name    string
	kind    string
	config  any
	path    string
	running bool
}

func TestImportSaveNodeRejectsInvalidProtocolFields(t *testing.T) {
	for _, test := range []importValidationCase{
		{name: "vless flow", kind: "vless", config: map[string]any{"flow": "unsupported"}, path: "config.flow"},
		{name: "vmess security", kind: "vmess", config: map[string]any{"security": "unsupported"}, path: "config.security"},
		{name: "tuic uuid missing", kind: "tuic", path: "config.uuid"},
		{name: "ss method", kind: "shadowsocks", path: "config.method"},
		{name: "ss password", kind: "shadowsocks", config: map[string]any{"method": "aes-128-gcm"}, path: "config.password"},
		{name: "ss secret key", kind: "shadowsocks", path: "config.password",
			config: map[string]any{"method": "2022-blake3-aes-128-gcm", "password": "private-test-key"}},
		{name: "raw must be object", kind: "vless", config: []string{"private-test-key"}, path: "config"},
		{name: "active kernel", kind: "tuic", path: "config.uuid", running: true},
		{name: "raw type override", kind: "vless", config: map[string]any{"type": "tuic"}, path: "config.uuid"},
	} {
		t.Run(test.name, func(t *testing.T) { assertImportRejectsNode(t, test) })
	}
}

func assertImportRejectsNode(t *testing.T, test importValidationCase) {
	t.Helper()
	fixture := newNodeSyncReloadFixture(t)
	beforeNodes := fixture.nodes.List()
	probe := &nodeSyncReloadProbe{running: test.running}
	svc := NewImportService(
		fixture.nodes,
		fixture.subs,
		fixture.path,
		probe,
	)
	err := svc.SaveNode(context.Background(), NodeInput{
		Tag: "desired-node", Type: test.kind, Server: "192.0.2.1", Port: 443, Config: test.config,
	})
	assertImportValidationError(t, err, test.path)
	if !reflect.DeepEqual(beforeNodes, fixture.nodes.List()) {
		t.Fatal("invalid input changed the node database")
	}
	after, err := os.ReadFile(fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fixture.previous, after) {
		t.Fatal("invalid input changed the active configuration")
	}
	kernelUnchanged := probe.running == test.running && probe.reloadCalls == 0 && probe.restartCalls == 0
	if !kernelUnchanged {
		t.Fatalf("invalid input changed kernel state: %#v", probe)
	}
}

func assertImportValidationError(t *testing.T, err error, path string) {
	t.Helper()
	var domain *DomainError
	if !errors.As(err, &domain) {
		t.Fatalf("want validation domain error, got %v", err)
	}
	if domain.Status != 400 || domain.Code != model.ErrorInvalidRequest {
		t.Fatalf("want invalid_request status 400, got %#v", domain)
	}
	if !strings.HasPrefix(domain.Message, path+":") {
		t.Fatalf("want field path %s, got %q", path, domain.Message)
	}
	if strings.Contains(domain.Message, "private-test-key") {
		t.Fatal("validation response disclosed credentials")
	}
}

func TestImportSaveNodePreservesSupportedCredentialsWhileStopped(t *testing.T) {
	const key = "AAAAAAAAAAAAAAAAAAAAAA=="
	for _, test := range []struct {
		name string
		kind string
		raw  any
	}{
		{name: "vless empty identity", kind: "vless"},
		{name: "vmess named identity", kind: "vmess", raw: map[string]any{"uuid": "named identity"}},
		{name: "ss none empty password", kind: "shadowsocks", raw: map[string]any{"method": "none"}},
		{name: "ss2022 key chain", kind: "shadowsocks", raw: map[string]any{"method": "2022-blake3-aes-128-gcm", "password": key + ":" + key}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newNodeSyncReloadFixture(t)
			instance := core.NewSBInstance(fixture.path, core.NewLogWriter(5))
			svc := NewImportService(
				fixture.nodes,
				fixture.subs,
				fixture.path,
				instance,
			)
			if err := svc.SaveNode(context.Background(), NodeInput{
				Tag: "accepted", Type: test.kind, Server: "192.0.2.1", Port: 443, Config: test.raw,
			}); err != nil {
				t.Fatal(err)
			}
			if fixture.nodes.Get("accepted") == nil || instance.Status().Running {
				t.Fatal("valid credentials must persist without starting a stopped kernel")
			}
		})
	}
}

func TestImportSaveNodePropagatesSyncFailure(t *testing.T) {
	fixture := newNodeSyncReloadFixture(t)
	probe := &nodeSyncReloadProbe{running: true, reloadErr: errors.New("kernel reload failed")}
	svc := NewImportService(
		fixture.nodes,
		fixture.subs,
		fixture.path,
		probe,
	)
	err := svc.SaveNode(context.Background(), NodeInput{
		Tag: "accepted", Type: "vless", Server: "192.0.2.1", Port: 443,
	})
	var domain *DomainError
	if !errors.As(err, &domain) {
		t.Fatalf("want synchronization domain error, got %v", err)
	}
	if domain.Status != 500 || domain.Code != model.ErrorNodeUpdateFailed {
		t.Fatalf("want node synchronization failure, got %#v", domain)
	}
	if fixture.nodes.Get("accepted") == nil {
		t.Fatal("valid node must remain available for synchronization retry")
	}
	after, err := os.ReadFile(fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fixture.previous, after) || !probe.running {
		t.Fatal("sync failure must restore the prior active configuration and kernel")
	}
}
