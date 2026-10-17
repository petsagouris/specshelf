package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/petsagouris/specshelf/internal/vault"
)

// fakeStore is a Store backed by memory and nothing else. It counts its calls so
// a test can prove the session filtered without touching the filesystem.
type fakeStore struct {
	records  []vault.IndexRecord
	warnings []vault.Warning
	err      error
	content  map[string]string
	findErr  error

	lists int
	finds int
}

func (f *fakeStore) List() ([]vault.IndexRecord, []vault.Warning, error) {
	f.lists++
	return f.records, f.warnings, f.err
}

func (f *fakeStore) Find(ref string) (vault.Found, error) {
	f.finds++
	if f.findErr != nil {
		return vault.Found{}, f.findErr
	}
	return vault.Found{Path: ref, Data: []byte(f.content[ref])}, nil
}

func sampleRecords() []vault.IndexRecord {
	return []vault.IndexRecord{
		{ID: "spec-20260102-alpha", Kind: "spec", Title: "Alpha spec", Status: "open", Updated: "2026-01-02", Repos: []string{"specshelf"}, Tags: []string{"work"}, Path: "specs/2026-01-02-alpha.md", Summary: "Alpha summary"},
		{ID: "decision-20260101-beta", Kind: "decision", Title: "Beta decision", Status: "accepted", Updated: "2026-01-01", Repos: []string{"infra"}, Tags: []string{"infra", "work"}, Path: "decisions/2026-01-01-beta.md", Summary: "Beta summary"},
		{ID: "spec-20251231-gamma", Kind: "spec", Title: "Gamma spec", Status: "done", Updated: "2025-12-31", Path: "specs/2025-12-31-gamma.md", Summary: "Gamma summary"},
	}
}

func sampleContent() map[string]string {
	return map[string]string{
		"spec-20260102-alpha": "---\nid: spec-20260102-alpha\nkind: spec\ntitle: Alpha spec\n---\n\n# Alpha spec\n\nBody line one.\nBody line two.\n",
	}
}

// newFake returns a session model over the sample records, already loaded.
func newFake() (*Model, *fakeStore) {
	store := &fakeStore{records: sampleRecords(), content: sampleContent()}
	m := New(store)
	m.Update(loadedMsg{records: store.records, warnings: store.warnings})
	return m, store
}

// k builds a key press whose String() is the key's text.
func k(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

// kc builds a special key press such as enter or tab.
func kc(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

var ctrlC = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}

// update feeds msg to m and returns the command it produced.
func update(m *Model, msg tea.Msg) tea.Cmd {
	_, cmd := m.Update(msg)
	return cmd
}

// run executes cmd and returns its message.
func execCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}
	return cmd()
}

func TestListMovesSelection(t *testing.T) {
	m, _ := newFake()
	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}
	update(m, k('j'))
	if m.cursor != 1 {
		t.Fatalf("after j cursor = %d, want 1", m.cursor)
	}
	update(m, kc(tea.KeyDown))
	if m.cursor != 2 {
		t.Fatalf("after down cursor = %d, want 2", m.cursor)
	}
	update(m, k('j'))
	if m.cursor != 2 {
		t.Fatalf("cursor past the end = %d, want 2", m.cursor)
	}
	update(m, k('k'))
	if m.cursor != 1 {
		t.Fatalf("after k cursor = %d, want 1", m.cursor)
	}
}

func TestOpenDetail(t *testing.T) {
	m, store := newFake()
	cmd := update(m, kc(tea.KeyEnter))
	if m.mode != modeDetail {
		t.Fatalf("mode = %d, want detail", m.mode)
	}
	msg := execCmd(t, cmd)
	update(m, msg)
	if store.finds != 1 {
		t.Fatalf("Find calls = %d, want 1", store.finds)
	}
	view := m.View().Content
	if !strings.Contains(view, "Body line one.") {
		t.Fatalf("detail view does not show the body:\n%s", view)
	}
	if !strings.Contains(view, "id: spec-20260102-alpha") {
		t.Fatalf("detail view does not show the frontmatter:\n%s", view)
	}
	update(m, kc(tea.KeyEsc))
	if m.mode != modeList {
		t.Fatalf("after esc mode = %d, want list", m.mode)
	}
}

func TestHelpAndBack(t *testing.T) {
	m, _ := newFake()
	update(m, k('?'))
	if m.mode != modeHelp {
		t.Fatalf("mode = %d, want help", m.mode)
	}
	if !strings.Contains(m.View().Content, "move the selection down") {
		t.Fatalf("help view is missing bindings:\n%s", m.View().Content)
	}
	update(m, kc(tea.KeyEsc))
	if m.mode != modeList {
		t.Fatalf("after esc mode = %d, want list", m.mode)
	}
}

func TestCreatePromptCapturesTitle(t *testing.T) {
	m, _ := newFake()
	update(m, k('c'))
	if m.mode != modeCreate {
		t.Fatalf("mode = %d, want create", m.mode)
	}
	for _, r := range "My spec" {
		update(m, k(r))
	}
	update(m, kc(tea.KeyTab))
	if m.createKind != "decision" {
		t.Fatalf("kind after tab = %q, want decision", m.createKind)
	}
	update(m, kc(tea.KeyEnter))
	if m.mode != modeList {
		t.Fatalf("after enter mode = %d, want list", m.mode)
	}
	if m.Create == nil {
		t.Fatal("enter did not capture a create request")
	}
	if m.Create.Title != "My spec" || m.Create.Kind != "decision" {
		t.Fatalf("captured %+v, want title %q kind decision", m.Create, "My spec")
	}
}

func TestCreateCancelLeavesNothing(t *testing.T) {
	m, _ := newFake()
	update(m, k('c'))
	update(m, k('x'))
	update(m, kc(tea.KeyEsc))
	if m.Create != nil {
		t.Fatalf("cancel captured a request: %+v", m.Create)
	}
	if m.mode != modeList {
		t.Fatalf("after esc mode = %d, want list", m.mode)
	}
}

func TestQuitKeys(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{k('q'), ctrlC} {
		m, _ := newFake()
		cmd := update(m, key)
		if !m.quitting {
			t.Fatalf("key %q did not mark the session quitting", key.String())
		}
		if msg := execCmd(t, cmd); msg != tea.Quit() {
			t.Fatalf("key %q produced %#v, want QuitMsg", key.String(), msg)
		}
	}
}
