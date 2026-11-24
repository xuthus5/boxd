package singbox

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/trafficcontrol"
	M "github.com/sagernet/sing/common/metadata"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/adapters/outbound/singboxconfig"
	"github.com/xuthus5/boxd/kernel"
)

const testDirectTag = "direct"

func TestRuntimeDelayModeDNSAndConnections(t *testing.T) {
	ctx := context.Background()
	state := control.DefaultDesiredState()
	state.Inbounds[0].Listen.Port = freeRuntimePort(t)
	state.Routes.Presets.ChinaDirect = false
	state.Routes.Presets.AdBlock = false
	document, err := singboxconfig.New().Compile(ctx, state)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	parsed, err := parseOptions(ctx, document, t.TempDir())
	if err != nil {
		t.Fatalf("parseOptions() error = %v", err)
	}
	engine := newEngine(kernelOpenOptions(t.TempDir(), document), parsed)
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = engine.Close() })

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	delay, err := engine.OutboundDelay(ctx, testDirectTag, target.URL)
	if err != nil {
		t.Fatalf("OutboundDelay() error = %v", err)
	}
	if delay < 0 {
		t.Fatalf("OutboundDelay() = %v", delay)
	}

	mode, err := engine.ClashMode(ctx)
	if err != nil || mode.Current == "" || len(mode.Available) == 0 {
		t.Fatalf("ClashMode() = %+v, %v", mode, err)
	}
	if err := engine.SetClashMode(ctx, mode.Current); err != nil {
		t.Fatalf("SetClashMode(current) error = %v", err)
	}
	if err := engine.SetClashMode(ctx, "not-a-mode"); !hasKernelCode(err, kernel.ErrCodeInvalidConfig) {
		t.Fatalf("SetClashMode(invalid) error = %v", err)
	}
	if err := engine.FlushDNS(ctx); err != nil {
		t.Fatalf("FlushDNS() error = %v", err)
	}

	connections, err := engine.ListConnections(ctx)
	if err != nil {
		t.Fatalf("ListConnections() error = %v", err)
	}
	if len(connections) != 0 {
		t.Fatalf("unexpected idle connections: %+v", connections)
	}
	if err := engine.CloseConnection(ctx, 1); !hasKernelCode(err, kernel.ErrCodeNotFound) {
		t.Fatalf("CloseConnection(missing) error = %v", err)
	}
	if count, err := engine.CloseConnections(ctx, kernel.ConnectionFilter{Outbound: testDirectTag}); err != nil || count != 0 {
		t.Fatalf("CloseConnections() = %d, %v", count, err)
	}
	traffic := engine.TrafficSnapshot()
	if traffic.At.IsZero() || traffic.UploadRate < 0 || traffic.DownloadRate < 0 {
		t.Fatalf("TrafficSnapshot() = %+v", traffic)
	}
}

func TestConnectionViewAndFilter(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	upload := new(atomic.Int64)
	download := new(atomic.Int64)
	upload.Store(12)
	download.Store(34)
	metadata := &trafficcontrol.TrackerMetadata{
		ID: id,
		Metadata: adapter.InboundContext{
			Inbound:     "inbound-mixed",
			InboundType: "mixed",
			Network:     "tcp",
			Source:      M.ParseSocksaddr("127.0.0.1:12345"),
			Destination: M.ParseSocksaddr("example.com:443"),
			Domain:      "example.com",
			ProcessInfo: &adapter.ConnectionOwner{ProcessPath: "/usr/bin/example"},
		},
		CreatedAt: time.Now().UTC(),
		Upload:    upload,
		Download:  download,
		Outbound:  testDirectTag,
	}
	view := connectionView(metadata, nil)
	if view.ID != connectionID(id) || view.Inbound != "mixed/inbound-mixed" || view.Target != "example.com" {
		t.Fatalf("connectionView() = %+v", view)
	}
	if view.Process != "/usr/bin/example" || view.Upload != 12 || view.Download != 34 || view.Rule != "final" {
		t.Fatalf("connectionView() = %+v", view)
	}
	if view.RuleIndex != -1 {
		t.Fatalf("connectionView() rule index = %d, want -1", view.RuleIndex)
	}
	if !connectionMatches(metadata, kernel.ConnectionFilter{Outbound: testDirectTag, Process: "/usr/bin/example"}, nil) {
		t.Fatal("connection did not match equivalent filter")
	}
	if connectionMatches(metadata, kernel.ConnectionFilter{Outbound: "other"}, nil) {
		t.Fatal("connection matched a different outbound")
	}
	ids := map[int64]struct{}{connectionID(id): {}}
	if !connectionMatches(metadata, kernel.ConnectionFilter{}, ids) {
		t.Fatal("connection did not match its ID filter")
	}
}

func hasKernelCode(err error, code kernel.ErrorCode) bool {
	var kernelError *kernel.Error
	return errors.As(err, &kernelError) && kernelError.Code == code
}

func freeRuntimePort(t *testing.T) uint16 {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer listener.Close()
	_, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split address: %v", err)
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return uint16(port)
}
