//go:build linux

package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	box "github.com/sagernet/sing-box"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

func TestMain(tests *testing.M) {
	if handled, err := RunInternalCommand(os.Args[1:], os.Stdin); handled {
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(tests.Run())
}

func TestRealBoxUnshareNamespaceLifecycle(t *testing.T) {
	originalNS, err := os.Stat("/proc/thread-self/ns/net")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "holder.pid")
	instance := newNamespaceRuntime(t, path)
	if err := instance.Start(); err != nil {
		if errors.Is(err, os.ErrPermission) {
			if code := ClassifyKernelError(err.Error(), err); code != KernelErrorPermission {
				t.Fatalf("namespace permission error classified as %q", code)
			}
			if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("failed startup leaked pid file: %v", statErr)
			}
			t.Skipf("unshare requires CAP_SYS_ADMIN: %v", err)
		}
		t.Fatal(err)
	}
	pid := checkNamespaceHolder(t, instance, path)
	holderNS, err := os.Stat(instance.namespaces.ResolvePath("audit"))
	if err != nil || os.SameFile(originalNS, holderNS) {
		t.Fatalf("holder must have a separate network namespace: %v", err)
	}
	if err := instance.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("shutdown leaked pid file: %v", err)
	}
	waitForNamespaceHolderExit(t, pid)
}

func checkNamespaceHolder(t *testing.T, instance realBox, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pid := strings.TrimSpace(string(content))
	if instance.namespaces.ResolvePath("audit") != "/proc/"+pid+"/ns/net" {
		t.Fatal("pid file does not identify the actual namespace")
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("pid file permissions: %v %v", info, err)
	}
	command, err := os.ReadFile("/proc/" + pid + "/cmdline")
	if err != nil || !strings.Contains(string(command), networkNamespaceHolderCommand) {
		t.Fatalf("namespace was not held by the internal helper: %v", err)
	}
	return pid
}

func waitForNamespaceHolderExit(t *testing.T, pid string) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for {
		if _, err := os.Stat("/proc/" + pid); errors.Is(err, os.ErrNotExist) {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatal("namespace holder did not exit when the parent pipe closed")
		}
	}
}

func TestRealBoxNamespaceStartPreservesExistingPIDFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "holder.pid")
	if err := os.WriteFile(path, []byte("owned by another process"), 0600); err != nil {
		t.Fatal(err)
	}
	instance := newNamespaceRuntime(t, path)
	if err := instance.Start(); !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected exclusive pid file error, got %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "owned by another process" {
		t.Fatalf("existing pid file was changed: %q %v", content, err)
	}
}

func newNamespaceRuntime(t *testing.T, path string) realBox {
	t.Helper()
	instance, err := newRealBox(box.Options{
		Context: include.Context(context.Background()),
		Options: option.Options{
			Log: &option.LogOptions{Disabled: true},
			NetworkNamespaces: []option.NetworkNamespace{{
				Type: C.NetNsTypeUnshare, Tag: "audit",
				UnshareOptions: option.UnshareNetworkNamespaceOptions{PidFile: path},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := instance.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			t.Errorf("close namespace runtime: %v", err)
		}
	})
	return instance
}

func TestNamespacePIDPublicationRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	path, moved := filepath.Join(dir, "holder.pid"), filepath.Join(dir, "moved.pid")
	files := namespacePIDFiles{{tag: "private", path: path}}
	if err := files.prepare(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = files.close() })
	if err := os.Rename(path, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(moved, path); err != nil {
		t.Fatal(err)
	}
	if err := files.publish(fakeNamespacePaths{"private": "/proc/123/ns/net"}); !errors.Is(err, errNamespacePIDReplaced) {
		t.Fatalf("expected symlink publication error, got %v", err)
	}
}
