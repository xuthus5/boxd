package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/xuthus5/boxui/internal/config"
)

func TestValidateConfigTLS(t *testing.T) {
	cert := filepath.Join(t.TempDir(), "cert.pem")
	key := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(cert, []byte("cert"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key, []byte("key"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Password: "pass",
		TLSCert:  cert,
		TLSKey:   key,
		LogLevel: "debug",
	}
	if err := validateConfig(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateConfigRejectsIncompleteTLS(t *testing.T) {
	if err := validateConfig(&config.Config{TLSCert: "/tmp/cert.pem"}); err == nil {
		t.Fatal("expected incomplete TLS configuration error")
	}
	if err := validateConfig(&config.Config{TLSKey: "/tmp/key.pem"}); err == nil {
		t.Fatal("expected incomplete TLS configuration error")
	}
}

func TestValidateConfigRejectsInvalidTLSFiles(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.pem")
	if err := validateConfig(&config.Config{TLSCert: missing, TLSKey: missing}); err == nil {
		t.Fatal("expected inaccessible certificate error")
	}
	cert := filepath.Join(t.TempDir(), "cert.pem")
	if err := os.WriteFile(cert, []byte("cert"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateConfig(&config.Config{TLSCert: cert, TLSKey: t.TempDir()}); err == nil {
		t.Fatal("expected invalid key file error")
	}
}

func TestRunTLSConfig(t *testing.T) {
	previousServer := makeHTTPServer
	previousSignal := makeSignalChannel
	t.Cleanup(func() {
		makeHTTPServer = previousServer
		makeSignalChannel = previousSignal
	})

	fake := &fakeServer{listenErr: http.ErrServerClosed}
	makeHTTPServer = func(addr string, handler http.Handler) server {
		return fake
	}
	makeSignalChannel = func() chan os.Signal {
		quit := make(chan os.Signal, 1)
		quit <- syscall.SIGTERM
		return quit
	}

	cert := filepath.Join(t.TempDir(), "cert.pem")
	key := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(cert, []byte("cert"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key, []byte("key"), 0600); err != nil {
		t.Fatal(err)
	}
	err := run(&config.Config{
		Listen:     "127.0.0.1:0",
		ConfigPath: filepath.Join(t.TempDir(), "sing-box.json"),
		DataDir:    t.TempDir(),
		Username:   "admin",
		Password:   "pass",
		TLSCert:    cert,
		TLSKey:     key,
		LogLevel:   "debug",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestServeUntilSignalTLS(t *testing.T) {
	quit := make(chan os.Signal, 1)
	quit <- syscall.SIGTERM

	fake := &fakeServer{listenErr: http.ErrServerClosed}
	cfg := &config.Config{
		Listen:  "127.0.0.1:0",
		TLSCert: "/path/cert.pem",
		TLSKey:  "/path/key.pem",
	}
	if err := serveUntilSignal(fake, cfg, quit); err != nil {
		t.Fatal(err)
	}
	if !fake.shutdownCalled {
		t.Fatal("shutdown should be called")
	}
}

func TestServeUntilSignalTLSError(t *testing.T) {
	quit := make(chan os.Signal)
	fake := &fakeServer{listenErr: errors.New("tls listen failed")}
	cfg := &config.Config{
		Listen:  "127.0.0.1:0",
		TLSCert: "/path/cert.pem",
		TLSKey:  "/path/key.pem",
	}
	err := serveUntilSignal(fake, cfg, quit)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenDatabaseExistingDir(t *testing.T) {
	dataDir := t.TempDir()
	db, err := openDatabase(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
}
