package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// loadedFake returns a model whose list came from the store's own List call, so
// the call count is meaningful.
func loadedFake(t *testing.T) (*Model, *fakeStore) {
	t.Helper()
	store := &fakeStore{records: sampleRecords(), content: sampleContent()}
	m := New(store)
	update(m, execCmd(t, m.Init()))
	return m, store
}

func TestFilterTextNarrowsWithoutFilesystem(t *testing.T) {
	m, store := loadedFake(t)
	if store.lists != 1 {
		t.Fatalf("List calls after load = %d, want 1", store.lists)
	}
	if got := len(m.visible()); got != 3 {
		t.Fatalf("visible before filtering = %d, want 3", got)
	}

	update(m, k('/'))
	for _, r := range "beta" {
		update(m, k(r))
	}
	if got := len(m.visible()); got != 1 {
		t.Fatalf("visible after query = %d, want 1", got)
	}
	if m.visible()[0].ID != "decision-20260101-beta" {
		t.Fatalf("query matched %q, want the beta decision", m.visible()[0].ID)
	}
	if store.lists != 1 {
		t.Fatalf("typing in the filter re-read the vault: %d List calls", store.lists)
	}

	// A second typed query narrows further, still without a read.
	for _, r := range "zzz" {
		update(m, k(r))
	}
	if got := len(m.visible()); got != 0 {
		t.Fatalf("visible after narrowing query = %d, want 0", got)
	}
	if store.lists != 1 {
		t.Fatalf("typing in the filter re-read the vault: %d List calls", store.lists)
	}

	// Reload is the only operation that re-reads.
	update(m, kc(tea.KeyEnter))
	execCmd(t, update(m, k('r')))
	if store.lists != 2 {
		t.Fatalf("List calls after reload = %d, want 2", store.lists)
	}
}

func TestFilterQueryMatchesMetadata(t *testing.T) {
	m, _ := loadedFake(t)
	for _, query := range []string{"alpha", "specs/2025", "infra", "open"} {
		m.filter = Filter{Query: query}
		if got := len(m.visible()); got == 0 {
			t.Fatalf("query %q matched nothing", query)
		}
	}
}

func TestFilterEscClearsQuery(t *testing.T) {
	m, _ := loadedFake(t)
	update(m, k('/'))
	update(m, k('b'))
	if len(m.visible()) == 3 {
		t.Fatal("query did not narrow the list")
	}
	update(m, kc(tea.KeyEsc))
	if m.filter.Query != "" {
		t.Fatalf("esc left query %q, want empty", m.filter.Query)
	}
	if len(m.visible()) != 3 {
		t.Fatalf("visible after esc = %d, want 3", len(m.visible()))
	}
}

func TestFilterCyclesKindStatusRepoTag(t *testing.T) {
	cases := []struct {
		name  string
		key   rune
		field func(*Model) string
		want  []string
	}{
		{"kind", 'K', func(m *Model) string { return m.filter.Kind }, []string{"spec", "decision", ""}},
		{"status", 'S', func(m *Model) string { return m.filter.Status }, []string{"open", "accepted", "done", ""}},
		{"repo", 'R', func(m *Model) string { return m.filter.Repo }, []string{"specshelf", "infra", ""}},
		{"tag", 'T', func(m *Model) string { return m.filter.Tag }, []string{"work", "infra", ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := loadedFake(t)
			counts := map[string]int{}
			for _, value := range tc.want {
				update(m, k(tc.key))
				if got := tc.field(m); got != value {
					t.Fatalf("after cycling %s filter = %q, want %q", tc.name, got, value)
				}
				counts[value] = len(m.visible())
			}
			if counts[""] != 3 {
				t.Fatalf("clearing the %s filter shows %d, want 3", tc.name, counts[""])
			}
			for _, value := range tc.want[:len(tc.want)-1] {
				if counts[value] >= 3 {
					t.Fatalf("%s=%q did not narrow the list (%d)", tc.name, value, counts[value])
				}
			}
		})
	}
}

func TestFilterKeysAreDocumented(t *testing.T) {
	want := map[string]bool{"/": true, "K": true, "S": true, "R": true, "T": true}
	for _, binding := range Bindings {
		for _, key := range binding.Keys {
			delete(want, key)
		}
	}
	if len(want) != 0 {
		t.Fatalf("filter keys missing from Bindings: %v", want)
	}
}
