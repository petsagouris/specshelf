package tui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
)

// The packages internal/tui must never import: the frontmatter codec and
// anything that would let it read or write a vault file without going through
// internal/vault.
var forbiddenImports = map[string]string{
	"github.com/petsagouris/specshelf/internal/frontmatter": "the frontmatter codec",
	"io/ioutil":     "filesystem access",
	"path/filepath": "filesystem access",
	"io/fs":         "filesystem access",
	"os/exec":       "process execution",
}

// The os selectors internal/tui may use. It may name the standard streams and
// the terminal-mode bit its guard needs; every file operation is refused.
var allowedOSNames = map[string]bool{
	"Stdout": true, "Stderr": true, "Stdin": true,
	"File": true, "ModeCharDevice": true,
}

// TestImportBoundary asserts internal/tui reaches the vault only through
// internal/vault and never touches a vault file itself.
func TestImportBoundary(t *testing.T) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	vaultImported := false
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		imports := importNames(file)
		for local, importPath := range imports {
			if why, ok := forbiddenImports[importPath]; ok {
				t.Errorf("%s imports %s: %s", name, importPath, why)
			}
			if importPath == "github.com/petsagouris/specshelf/internal/vault" {
				vaultImported = true
			}
			if importPath == "os" {
				checkOSNames(t, fset, file, local)
			}
		}
	}
	if !vaultImported {
		t.Error("internal/tui does not import internal/vault; every vault read and mutation must go through it")
	}
}

// importNames maps each import's local name to its path.
func importNames(file *ast.File) map[string]string {
	names := map[string]string{}
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		local := path.Base(importPath)
		if spec.Name != nil {
			local = spec.Name.Name
		}
		names[local] = importPath
	}
	return names
}

// checkOSNames fails on any os selector that is not a permitted name, which is
// what catches an added os.ReadFile or os.Open.
func checkOSNames(t *testing.T, fset *token.FileSet, file *ast.File, local string) {
	t.Helper()
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if !ok || ident.Name != local {
			return true
		}
		if !allowedOSNames[selector.Sel.Name] {
			position := fset.Position(selector.Pos())
			t.Errorf("%s: os.%s is not allowed in internal/tui; vault files are reached through internal/vault", position, selector.Sel.Name)
		}
		return true
	})
}
