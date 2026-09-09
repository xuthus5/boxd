package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

func TestNamespacePIDOptionsPreserveCallerConfiguration(t *testing.T) {
	input := []option.NetworkNamespace{
		{Type: C.NetNsTypeDefault, Tag: "existing"},
		{Type: C.NetNsTypeUnshare, Tag: "without-file"},
		{Type: C.NetNsTypeUnshare, Tag: "private", UnshareOptions: option.UnshareNetworkNamespaceOptions{PidFile: "holder.pid"}},
	}
	options, files := takeNamespacePIDFiles(input)
	if len(files) != 1 || files[0].tag != "private" || files[0].path != "holder.pid" {
		t.Fatalf("unexpected pid requests: %+v", files)
	}
	if options[2].UnshareOptions.PidFile != "" || input[2].UnshareOptions.PidFile != "holder.pid" {
		t.Fatal("runtime must take ownership without mutating saved configuration")
	}
}

func TestNamespacePIDFilesPublishPrivatelyAndCleanUp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "holder.pid")
	files := namespacePIDFiles{{tag: "private", path: path}}
	if err := files.prepare(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = files.close() })
	if err := files.publish(fakeNamespacePaths{"private": "/proc/123/ns/net"}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "123\n" || info.Mode().Perm() != 0600 {
		t.Fatalf("content=%q permissions=%o", content, info.Mode().Perm())
	}
	if err := files.close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owned pid file survived shutdown: %v", err)
	}
}

func TestNamespacePIDFilesRejectExistingAndRollbackPartialPreparation(t *testing.T) {
	dir := t.TempDir()
	owned := filepath.Join(dir, "owned.pid")
	existing := filepath.Join(dir, "existing.pid")
	if err := os.WriteFile(existing, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	files := namespacePIDFiles{{tag: "first", path: owned}, {tag: "second", path: existing}}
	if err := files.prepare(); !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected existing-path failure, got %v", err)
	}
	if _, err := os.Lstat(owned); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial startup left pid file: %v", err)
	}
	content, err := os.ReadFile(existing)
	if err != nil || string(content) != "keep" {
		t.Fatalf("existing pid file changed: %q %v", content, err)
	}
}

func TestNamespacePIDFilesPreserveReplacementAndHandleExternalRemoval(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		t.Run(map[bool]string{false: "removed", true: "replaced"}[replacement], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "holder.pid")
			files := namespacePIDFiles{{tag: "private", path: path}}
			if err := files.prepare(); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if replacement {
				if err := os.WriteFile(path, []byte("other owner"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := files.close(); err != nil {
				t.Fatal(err)
			}
			if replacement {
				content, err := os.ReadFile(path)
				if err != nil || string(content) != "other owner" {
					t.Fatalf("replacement was removed: %q %v", content, err)
				}
			}
		})
	}
}

func TestNamespacePIDFilesRejectInvalidHolderPaths(t *testing.T) {
	for _, path := range []string{"private", "/proc/0/ns/net", "/proc/-1/ns/net", "/proc/0123/ns/net", "/proc/self/ns/net"} {
		t.Run(path, func(t *testing.T) {
			files := namespacePIDFiles{{tag: "private", path: filepath.Join(t.TempDir(), "holder.pid")}}
			if err := files.prepare(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = files.close() })
			if err := files.publish(fakeNamespacePaths{"private": path}); err == nil {
				t.Fatal("expected invalid holder path error")
			}
		})
	}
}

func TestNamespacePIDFilesSurfaceInitializationAndWriteErrors(t *testing.T) {
	files := namespacePIDFiles{{tag: "private", path: filepath.Join(t.TempDir(), "holder.pid")}}
	if err := files.publish(nil); err == nil {
		t.Fatal("expected uninitialized pid file error")
	}
	if err := files.prepare(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = files.close() })
	if err := files.publish(nil); err == nil {
		t.Fatal("expected missing namespace manager error")
	}
	if err := files[0].file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := files.publish(fakeNamespacePaths{"private": "/proc/123/ns/net"}); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected write failure, got %v", err)
	}
}

func TestNamespacePIDFilesSurfaceCleanupErrors(t *testing.T) {
	dir := t.TempDir()
	files := namespacePIDFiles{{tag: "private", path: filepath.Join(dir, "holder.pid")}}
	if err := files.prepare(); err != nil {
		t.Fatal(err)
	}
	if err := files[0].file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := files.close(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected descriptor close error, got %v", err)
	}
	files = namespacePIDFiles{{tag: "private", path: filepath.Join(dir, "holder.pid")}}
	if err := files.prepare(); err != nil {
		t.Fatal(err)
	}
	files[0].path = filepath.Join(files[0].path, "not-a-directory")
	if err := files.close(); err == nil {
		t.Fatal("expected stat failure while cleaning pid file")
	}
}

func TestNamespacePIDPublicationRejectsReplacedPath(t *testing.T) {
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
	if err := os.WriteFile(path, []byte("another process\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := files.publish(fakeNamespacePaths{"private": "/proc/123/ns/net"}); err == nil {
		t.Error("publishing to an inode no longer at pid_file must fail")
	}
	for _, entry := range []struct{ path, expected string }{{path, "another process\n"}, {moved, ""}} {
		content, err := os.ReadFile(entry.path)
		if err != nil || string(content) != entry.expected {
			t.Fatalf("publication changed %s: %q %v", entry.path, content, err)
		}
	}
}

func TestNamespacePIDPublicationRejectsRemovalAndPermissions(t *testing.T) {
	for _, removed := range []bool{true, false} {
		files := namespacePIDFiles{{tag: "private", path: filepath.Join(t.TempDir(), "holder.pid")}}
		if err := files.prepare(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = files.close() })
		var err error
		if removed {
			err = os.Remove(files[0].path)
		} else {
			err = os.Chmod(files[0].path, 0644)
		}
		if err != nil {
			t.Fatal(err)
		}
		err = files.publish(fakeNamespacePaths{"private": "/proc/123/ns/net"})
		if removed && !errors.Is(err, os.ErrNotExist) || !removed && !errors.Is(err, errNamespacePIDReplaced) {
			t.Fatalf("expected changed pid_file failure: removed=%v err=%v", removed, err)
		}
	}
}

type fakeNamespacePaths map[string]string

func (p fakeNamespacePaths) ResolvePath(name string) string { return p[name] }

var _ adapter.NetworkNamespaceManager = fakeNamespacePaths{}
