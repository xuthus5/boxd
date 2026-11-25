package architecture_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/xuthus5/boxd"

func TestPublicPackagesDoNotImportInternal(t *testing.T) {
	root := repositoryRoot(t)
	for _, directory := range []string{"kernel", "control"} {
		walkGoFiles(t, filepath.Join(root, directory), func(path string, imports []string) {
			for _, imported := range imports {
				if strings.HasPrefix(imported, modulePath+"/internal") {
					t.Errorf("%s imports internal package %s", relative(root, path), imported)
				}
			}
		})
	}
}

func TestSensitiveDriversStayInTheirAdapters(t *testing.T) {
	root := repositoryRoot(t)
	walkGoFiles(t, root, func(path string, imports []string) {
		relativePath := filepath.ToSlash(relative(root, path))
		for _, imported := range imports {
			if (imported == "github.com/sagernet/sing-box" || strings.HasPrefix(imported, "github.com/sagernet/sing-box/")) &&
				!strings.HasPrefix(relativePath, "internal/driver/singbox/") {
				t.Errorf("%s imports sing-box outside its driver", relativePath)
			}
			if imported == "modernc.org/sqlite" && !strings.HasPrefix(relativePath, "internal/adapters/outbound/sqlite/") {
				t.Errorf("%s imports SQLite driver outside its adapter", relativePath)
			}
		}
	})
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func walkGoFiles(t *testing.T, root string, visit func(path string, imports []string)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "webui" || entry.Name() == ".idea" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		imports := make([]string, 0, len(file.Imports))
		for _, spec := range file.Imports {
			imported, unquoteErr := strconv.Unquote(spec.Path.Value)
			if unquoteErr != nil {
				return unquoteErr
			}
			imports = append(imports, imported)
		}
		visit(path, imports)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
}

func relative(root, path string) string {
	result, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return result
}

func TestBusinessCoreDoesNotDependOnHostAdapters(t *testing.T) {
	root := repositoryRoot(t)
	for _, directory := range []string{"control", "internal/controlplane"} {
		walkGoFiles(t, filepath.Join(root, directory), func(path string, imports []string) {
			relativePath := filepath.ToSlash(relative(root, path))
			for _, imported := range imports {
				switch {
				case strings.HasPrefix(imported, modulePath+"/internal/adapters/inbound"):
					t.Errorf("%s imports inbound adapter %s", relativePath, imported)
				case imported == "github.com/xuthus5/boxd/internal/webauth":
					t.Errorf("%s imports browser authentication", relativePath)
				case strings.HasPrefix(imported, modulePath+"/internal/adapters/outbound/sqlite"):
					t.Errorf("%s imports the SQLite adapter directly", relativePath)
				case strings.HasPrefix(imported, "github.com/egoist/mygo"):
					t.Errorf("%s imports the desktop toolkit", relativePath)
				}
			}
		})
	}
}

func TestInboundAdaptersStayIndependent(t *testing.T) {
	root := repositoryRoot(t)
	walkGoFiles(t, filepath.Join(root, "internal/adapters/inbound"), func(path string, imports []string) {
		relativePath := filepath.ToSlash(relative(root, path))
		for _, imported := range imports {
			if !strings.HasPrefix(imported, modulePath+"/internal/adapters/inbound/") {
				continue
			}
			current := inboundAdapterName(relativePath)
			target := strings.TrimPrefix(imported, modulePath+"/internal/adapters/inbound/")
			if separator := strings.IndexByte(target, '/'); separator >= 0 {
				target = target[:separator]
			}
			if current != target {
				t.Errorf("%s imports another inbound adapter %s", relativePath, imported)
			}
		}
	})
}

func TestDesktopAdapterDoesNotImportWebOrBrowserAuth(t *testing.T) {
	root := repositoryRoot(t)
	directory := filepath.Join(root, "internal/adapters/inbound/mygo")
	walkGoFiles(t, directory, func(path string, imports []string) {
		relativePath := filepath.ToSlash(relative(root, path))
		for _, imported := range imports {
			switch {
			case strings.HasPrefix(imported, modulePath+"/internal/adapters/inbound/httpapi"):
				t.Errorf("%s imports the HTTP adapter", relativePath)
			case imported == "github.com/xuthus5/boxd/internal/webauth":
				t.Errorf("%s imports browser authentication", relativePath)
			case strings.HasPrefix(imported, modulePath+"/internal/adapters/outbound/sqlite"):
				t.Errorf("%s imports SQLite", relativePath)
			case strings.HasPrefix(imported, modulePath+"/internal/driver/singbox"):
				t.Errorf("%s imports the sing-box driver", relativePath)
			case imported == "net/http":
				t.Errorf("%s imports an HTTP server stack", relativePath)
			case imported == "os/exec":
				t.Errorf("%s imports process execution", relativePath)
			}
		}
	})
}

func TestDesktopCompositionRootKeepsWebIngressOut(t *testing.T) {
	root := repositoryRoot(t)
	directory := filepath.Join(root, "cmd", "boxd-desktop")
	walkGoFiles(t, directory, func(path string, imports []string) {
		relativePath := filepath.ToSlash(relative(root, path))
		for _, imported := range imports {
			switch {
			case strings.HasPrefix(imported, modulePath+"/internal/adapters/inbound/httpapi"):
				t.Errorf("%s imports the Web ingress", relativePath)
			case imported == "github.com/xuthus5/boxd/internal/webauth":
				t.Errorf("%s imports browser authentication", relativePath)
			case imported == "net/http":
				t.Errorf("%s imports an HTTP server stack", relativePath)
			case imported == "os/exec":
				t.Errorf("%s imports process execution", relativePath)
			}
		}
	})
}

func TestBusinessCoreHasNoClientSpecificBranches(t *testing.T) {
	root := repositoryRoot(t)
	for _, directory := range []string{"control", "internal/controlplane", "internal/adapters/inbound/mygo", "cmd/boxd-desktop"} {
		walkGoFiles(t, filepath.Join(root, directory), func(path string, imports []string) {
			_ = imports
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			lower := strings.ToLower(string(body))
			for _, forbidden := range []string{"hostkind", "clientkind", "fordesktop", "user-agent", "requestorigin"} {
				if strings.Contains(lower, forbidden) {
					t.Errorf("%s contains client-specific business branch %q", relative(root, path), forbidden)
				}
			}
		})
	}
}

func inboundAdapterName(relativePath string) string {
	const prefix = "internal/adapters/inbound/"
	value := strings.TrimPrefix(filepath.ToSlash(relativePath), prefix)
	if separator := strings.IndexByte(value, '/'); separator >= 0 {
		return value[:separator]
	}
	return value
}
