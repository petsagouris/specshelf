package tui

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var updateGolden = flag.Bool("update", false, "rewrite the snapshot golden files")

// listModel returns a loaded session sized to w×h.
func listModel(w, h int) *Model {
	m, _ := newFake()
	update(m, tea.WindowSizeMsg{Width: w, Height: h})
	return m
}

func detailModel(t *testing.T, w, h int) *Model {
	t.Helper()
	m := listModel(w, h)
	update(m, execCmd(t, update(m, kc(tea.KeyEnter))))
	return m
}

func createModel(w, h int) *Model {
	m := listModel(w, h)
	update(m, k('c'))
	for _, r := range "A new note" {
		update(m, k(r))
	}
	return m
}

func helpModel(w, h int) *Model {
	m := listModel(w, h)
	update(m, k('?'))
	return m
}

var snapshotSizes = []struct {
	name string
	w, h int
}{
	{"80x24", 80, 24},
	{"40x12", 40, 12},
	{"120x40", 120, 40},
}

// TestSnapshotList pins the list view at three terminal sizes.
func TestSnapshotList(t *testing.T) {
	for _, size := range snapshotSizes {
		size := size
		t.Run(size.name, func(t *testing.T) {
			checkGolden(t, "list_"+size.name, listModel(size.w, size.h).View().Content)
		})
	}
}

// TestSnapshotViews pins the detail, create and help views at the default size.
func TestSnapshotViews(t *testing.T) {
	cases := map[string]*Model{
		"detail_80x24": detailModel(t, 80, 24),
		"create_80x24": createModel(80, 24),
		"help_80x24":   helpModel(80, 24),
	}
	for name, model := range cases {
		t.Run(name, func(t *testing.T) {
			checkGolden(t, name, model.View().Content)
		})
	}
}

// checkGolden compares got with testdata/name.golden, rewriting the file when
// -update is set or it does not exist yet.
func checkGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run with -update to create it): %v", err)
	}
	if string(want) != got {
		t.Fatalf("snapshot %s differs (run with -update to rewrite)\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
	}
}
