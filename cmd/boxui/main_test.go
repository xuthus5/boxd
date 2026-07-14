package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxui/internal/config"
	"github.com/xuthus5/boxui/internal/core"
)

func TestWriteVersion(t *testing.T) {
	previous := core.Version
	core.Version = "v1.2.3-test"
	t.Cleanup(func() { core.Version = previous })
	var output bytes.Buffer
	if err := writeVersion(&output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "v1.2.3-test\n" {
		t.Fatalf("version output = %q", output.String())
	}
}

func TestExecutePrintsVersion(t *testing.T) {
	previous := core.Version
	core.Version = "v2.0.0-test"
	t.Cleanup(func() { core.Version = previous })
	var output bytes.Buffer
	if err := execute(&config.Config{ShowVersion: true}, &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "v2.0.0-test\n" {
		t.Fatalf("version output = %q", output.String())
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestExecuteReturnsVersionWriteError(t *testing.T) {
	if err := execute(&config.Config{ShowVersion: true}, failingWriter{}); err == nil {
		t.Fatal("expected version write error")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
		want string
	}{
		{
			name: "missing password uses default",
			cfg:  &config.Config{},
			want: "",
		},
		{
			// JWT 密钥不再来自环境变量或配置：启动时从数据库加载，缺失则随机生成。
			name: "valid without jwt secret",
			cfg:  &config.Config{Password: "pass"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.cfg)
			if tt.want == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)) {
				t.Fatalf("error = %v, want contains %q", err, tt.want)
			}
		})
	}
}

func TestRunReturnsDataDirectoryError(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(parent, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	err := execute(&config.Config{DataDir: filepath.Join(parent, "child")}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "failed to create data dir") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunStartsAndStopsWithInjectedServer(t *testing.T) {
	previousServer := makeHTTPServer
	previousSignal := makeSignalChannel
	t.Cleanup(func() {
		makeHTTPServer = previousServer
		makeSignalChannel = previousSignal
	})

	fake := &fakeServer{listenErr: http.ErrServerClosed}
	makeHTTPServer = func(addr string, handler http.Handler) server {
		if addr != "127.0.0.1:0" {
			t.Fatalf("addr = %q", addr)
		}
		return fake
	}
	makeSignalChannel = func() chan os.Signal {
		quit := make(chan os.Signal, 1)
		quit <- syscall.SIGTERM
		return quit
	}

	err := run(&config.Config{
		Listen:     "127.0.0.1:0",
		ConfigPath: filepath.Join(t.TempDir(), "sing-box.json"),
		DataDir:    t.TempDir(),
		Username:   "admin",
		Password:   "pass",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !fake.shutdownCalled {
		t.Fatal("shutdown should be called")
	}
}

func TestRunBackupAndRestoreOperations(t *testing.T) {
	sourceDir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "sing-box.json")
	if err := os.WriteFile(configPath, []byte(`{"log":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	db, err := openDatabase(sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	settings := core.NewSettingsManager(db)
	if err := settings.Set("backup_test", "present"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := run(&config.Config{
		DataDir:    sourceDir,
		ConfigPath: configPath,
		Username:   "admin",
		BackupPath: archive,
	}); err != nil {
		t.Fatalf("backup run error = %v", err)
	}

	targetDir := filepath.Join(t.TempDir(), "restored")
	targetConfig := filepath.Join(t.TempDir(), "config", "sing-box.json")
	if err := run(&config.Config{
		DataDir:     targetDir,
		ConfigPath:  targetConfig,
		RestorePath: archive,
	}); err != nil {
		t.Fatalf("restore run error = %v", err)
	}
	restored, err := openDatabase(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = restored.Close() }()
	if got := core.NewSettingsManager(restored).Get("backup_test"); got != "present" {
		t.Fatalf("restored value = %q", got)
	}
}

func TestValidateRegularFileErrors(t *testing.T) {
	if err := validateRegularFile(filepath.Join(t.TempDir(), "missing"), "test file"); err == nil {
		t.Fatal("expected missing file error")
	}
	if err := validateRegularFile(t.TempDir(), "test file"); err == nil {
		t.Fatal("expected directory error")
	}
}

func TestRunBackupAndRestoreErrors(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	err := run(&config.Config{
		DataDir:    t.TempDir(),
		ConfigPath: filepath.Join(t.TempDir(), "missing.json"),
		Username:   "admin",
		BackupPath: filepath.Join(blocked, "backup.tar.gz"),
	})
	if err == nil || !strings.Contains(err.Error(), "create backup") {
		t.Fatalf("backup error = %v", err)
	}

	err = run(&config.Config{
		DataDir:     t.TempDir(),
		ConfigPath:  filepath.Join(t.TempDir(), "config.json"),
		RestorePath: filepath.Join(t.TempDir(), "missing.tar.gz"),
	})
	if err == nil || !strings.Contains(err.Error(), "restore backup") {
		t.Fatalf("restore error = %v", err)
	}
}

func TestCheckReadinessRejectsMissingState(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "empty.db")
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := checkReadiness(db, filepath.Join(t.TempDir(), "config.json")); err == nil {
		t.Fatal("database without settings bucket should not be ready")
	}

	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("settings"))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := checkReadiness(db, filepath.Join(t.TempDir(), "missing", "config.json")); err == nil {
		t.Fatal("missing config directory should not be ready")
	}
}

func TestOpenDatabaseCreatesPrivateDataFiles(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	db, err := openDatabase(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	dirInfo, err := os.Stat(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0700 {
		t.Fatalf("dir mode = %o, want 0700", dirInfo.Mode().Perm())
	}

	dbInfo, err := os.Stat(filepath.Join(dataDir, "boxui.db"))
	if err != nil {
		t.Fatal(err)
	}
	if dbInfo.Mode().Perm() != 0600 {
		t.Fatalf("db mode = %o, want 0600", dbInfo.Mode().Perm())
	}
}

func TestOpenDatabaseReturnsCreateDirError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}

	db, err := openDatabase(filepath.Join(path, "child"))
	if err == nil {
		if db != nil {
			_ = db.Close()
		}
		t.Fatal("expected error")
	}
}

func TestNewHandlerHealth(t *testing.T) {
	dataDir := t.TempDir()
	db, err := openDatabase(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	settingsMgr := core.NewSettingsManager(db)
	handler, _, _ := newHandler(&config.Config{
		Listen:     "127.0.0.1:0",
		ConfigPath: filepath.Join(t.TempDir(), "sing-box.json"),
		DataDir:    dataDir,
		Username:   "admin",
		Password:   "pass",
	}, db, settingsMgr)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("health status = %d", rr.Code)
	}
}

func TestCheckReadiness(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	settings := core.NewSettingsManager(db)
	if _, err := settings.EnsureAdminCredential("admin", ""); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "sing-box.json")
	if err := checkReadiness(db, configPath); err != nil {
		t.Fatalf("healthy readiness error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := checkReadiness(db, configPath); err == nil {
		t.Fatal("closed database should not be ready")
	}
}

func TestNewHTTPServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	server := newHTTPServer("127.0.0.1:0", handler)
	httpServer, ok := server.(*http.Server)
	if !ok {
		t.Fatalf("server type = %T", server)
	}
	if httpServer.Addr != "127.0.0.1:0" || httpServer.Handler == nil {
		t.Fatalf("server = %#v", httpServer)
	}
	if httpServer.ReadHeaderTimeout != 10*time.Second || httpServer.ReadTimeout != 30*time.Second {
		t.Fatalf("read timeouts = %s / %s", httpServer.ReadHeaderTimeout, httpServer.ReadTimeout)
	}
	if httpServer.WriteTimeout != 0 || httpServer.IdleTimeout != 120*time.Second {
		t.Fatalf("write/idle timeouts = %s / %s", httpServer.WriteTimeout, httpServer.IdleTimeout)
	}
	if httpServer.MaxHeaderBytes != 1<<20 {
		t.Fatalf("MaxHeaderBytes = %d", httpServer.MaxHeaderBytes)
	}
}

func TestSignalChannel(t *testing.T) {
	ch := signalChannel()
	if ch == nil {
		t.Fatal("signal channel should not be nil")
	}
	signal.Stop(ch)
	close(ch)
}

func TestServeUntilSignal(t *testing.T) {
	quit := make(chan os.Signal, 1)
	quit <- syscall.SIGTERM

	server := &fakeServer{listenErr: http.ErrServerClosed}
	if err := serveUntilSignal(server, &config.Config{Listen: "127.0.0.1:0"}, quit); err != nil {
		t.Fatal(err)
	}
	if !server.shutdownCalled {
		t.Fatal("shutdown should be called")
	}
}

func TestServeUntilSignalReturnsListenError(t *testing.T) {
	quit := make(chan os.Signal)
	server := &fakeServer{listenErr: errors.New("listen failed")}
	err := serveUntilSignal(server, &config.Config{Listen: "127.0.0.1:0"}, quit)
	if err == nil || !strings.Contains(err.Error(), "listen failed") {
		t.Fatalf("error = %v", err)
	}
}

func TestServeUntilSignalReturnsShutdownError(t *testing.T) {
	quit := make(chan os.Signal, 1)
	quit <- syscall.SIGTERM

	server := &fakeServer{
		listenErr:   http.ErrServerClosed,
		shutdownErr: errors.New("shutdown failed"),
	}
	err := serveUntilSignal(server, &config.Config{Listen: "127.0.0.1:0"}, quit)
	if err == nil || !strings.Contains(err.Error(), "shutdown failed") {
		t.Fatalf("error = %v", err)
	}
}

type fakeServer struct {
	listenErr      error
	shutdownErr    error
	shutdownCalled bool
}

func (s *fakeServer) ListenAndServe() error {
	return s.listenErr
}

func (s *fakeServer) ListenAndServeTLS(certFile, keyFile string) error {
	return s.listenErr
}

func (s *fakeServer) Shutdown(ctx context.Context) error {
	s.shutdownCalled = true
	return s.shutdownErr
}
