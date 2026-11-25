package architecture_test

import (
	"go/parser"
	"go/token"
	"io/fs"
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
