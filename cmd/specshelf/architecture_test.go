package main

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestArchitecturePackageList asserts that the package table in
// docs/ARCHITECTURE.md is exactly the set of package directories on disk.
// Adding or removing a package without updating the document fails here, and so
// does documenting a package that has no directory.
func TestArchitecturePackageList(t *testing.T) {
	root := repoRoot(t)

	doc, err := os.ReadFile(filepath.Join(root, "docs", "ARCHITECTURE.md"))
	if err != nil {
		t.Fatal(err)
	}
	documented := documentedPackages(t, string(doc))

	onDisk := packageDirs(t, root, "internal")
	onDisk = append(onDisk, "cmd/specshelf")

	comparePackages(t, documented, onDisk)
}

// repoRoot walks up from the test's working directory to the directory holding
// go.mod, so the test needs no absolute path.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test's working directory")
		}
		dir = parent
	}
}

// documentedPackages returns the first column of the package table, between the
// document's "## Packages" heading and the next heading.
func documentedPackages(t *testing.T, doc string) []string {
	t.Helper()
	var pkgs []string
	inTable := false
	for _, line := range strings.Split(doc, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case !inTable && line == "## Packages":
			inTable = true
		case inTable && strings.HasPrefix(line, "## "):
			return pkgs
		case inTable && strings.HasPrefix(line, "| `"):
			pkgs = append(pkgs, strings.Trim(strings.SplitN(line, "|", 3)[1], " `"))
		}
	}
	if !inTable {
		t.Fatal(`docs/ARCHITECTURE.md has no "## Packages" section`)
	}
	if len(pkgs) == 0 {
		t.Fatal("docs/ARCHITECTURE.md has an empty package table")
	}
	return pkgs
}

// packageDirs lists the immediate subdirectories of rel under root, as
// slash-separated paths, the form the document uses.
func packageDirs(t *testing.T, root, rel string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, path.Join(rel, e.Name()))
		}
	}
	return dirs
}

// comparePackages reports each path in exactly one of the two sets, naming
// whether the document is stale or the table is missing a package.
func comparePackages(t *testing.T, documented, onDisk []string) {
	t.Helper()
	inDoc := make(map[string]bool, len(documented))
	for _, p := range documented {
		inDoc[p] = true
	}
	onDiskSet := make(map[string]bool, len(onDisk))
	for _, p := range onDisk {
		onDiskSet[p] = true
	}

	for _, p := range sortedKeys(inDoc) {
		if !onDiskSet[p] {
			t.Errorf("docs/ARCHITECTURE.md documents %q, which is not a package directory on disk", p)
		}
	}
	for _, p := range sortedKeys(onDiskSet) {
		if !inDoc[p] {
			t.Errorf("package directory %q is missing from the table in docs/ARCHITECTURE.md", p)
		}
	}
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
