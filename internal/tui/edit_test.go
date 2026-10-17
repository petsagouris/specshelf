package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/petsagouris/specshelf/internal/vault"
)

// fakeEditor is a vault.Editor a test injects in place of the real process. A
// nil edit behaves like the user closing the editor without saving.
type fakeEditor struct {
	calls int
	edit  func(path string) error
}

func (f *fakeEditor) Edit(path string) error {
	f.calls++
	if f.edit == nil {
		return nil
	}
	return f.edit(path)
}

// today is the local date the store stamps, in the form the frontmatter uses.
func today() string { return time.Now().Format("2006-01-02") }

// seedVault creates a temporary vault holding one document and returns the
// vault root, the document's id (the reference the session uses) and its path.
func seedVault(t *testing.T, title string) (root, ref, path string) {
	t.Helper()
	root = t.TempDir()
	path, err := vault.Create(root, vault.NewDocument{Kind: "spec", Title: title})
	if err != nil {
		t.Fatalf("seed %q: %v", title, err)
	}
	records, _, err := vault.List(root)
	if err != nil {
		t.Fatalf("list seeded vault: %v", err)
	}
	for _, record := range records {
		if record.Title == title {
			return root, record.ID, path
		}
	}
	t.Fatalf("seeded document %q not found", title)
	return "", "", ""
}

// backdate replaces a freshly created document's updated date with an old one,
// so an edit that changes nothing is distinguishable from one that stamps
// today. It returns the document's bytes after the rewrite.
func backdate(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	aged := strings.Replace(string(data), "updated: "+today(), "updated: 2000-01-01", 1)
	if aged == string(data) {
		t.Fatalf("document has no updated line for %s:\n%s", today(), data)
	}
	if err := os.WriteFile(path, []byte(aged), 0o644); err != nil {
		t.Fatalf("backdate %s: %v", path, err)
	}
	return []byte(aged)
}

// TestEditUnchangedLeavesBytesAndUpdated is the 5.3 unchanged case: an editor
// that saves nothing leaves the document and its updated field exactly as they
// were.
func TestEditUnchangedLeavesBytesAndUpdated(t *testing.T) {
	root, ref, path := seedVault(t, "Unchanged edit")
	before := backdate(t, path)
	editor := &fakeEditor{}
	store := VaultStore{Root: root, Editor: editor}

	if _, err := store.Edit(ref); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if editor.calls != 1 {
		t.Fatalf("editor calls = %d, want 1", editor.calls)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after edit: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("an unchanged edit rewrote the document\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if strings.Contains(string(after), "updated: "+today()) {
		t.Fatalf("an unchanged edit stamped updated:\n%s", after)
	}
}

// TestEditChangedStampsUpdatedAndRefreshesIndex is the 5.3 changed case: an
// editor that saves new bytes has the document re-read, updated stamped and the
// index refreshed.
func TestEditChangedStampsUpdatedAndRefreshesIndex(t *testing.T) {
	root, ref, path := seedVault(t, "Changed edit")
	backdate(t, path)
	editor := &fakeEditor{edit: func(p string) error {
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		data = append(data, []byte("An edited line.\n")...)
		return os.WriteFile(p, data, 0o644)
	}}
	store := VaultStore{Root: root, Editor: editor}

	if _, err := store.Edit(ref); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after edit: %v", err)
	}
	if !strings.Contains(string(after), "An edited line.") {
		t.Fatalf("the edited body is missing:\n%s", after)
	}
	if !strings.Contains(string(after), "updated: "+today()) {
		t.Fatalf("a changed edit did not stamp updated:\n%s", after)
	}
	index, err := os.ReadFile(filepath.Join(root, "index.jsonl"))
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	if !strings.Contains(string(index), `"updated": "`+today()+`"`) {
		t.Fatalf("the index was not refreshed after the edit:\n%s", index)
	}
}

// TestDetailEditKeyRoutesThroughStore is 5.3 at the session level: `e` in the
// detail view calls the store's Edit exactly once and the result reloads the
// list and re-reads the document.
func TestDetailEditKeyRoutesThroughStore(t *testing.T) {
	m, store := newFake()
	update(m, execCmd(t, update(m, kc(tea.KeyEnter))))

	cmd := update(m, k('e'))
	if cmd == nil {
		t.Fatal("`e` produced no command")
	}
	msg := execCmd(t, cmd)
	if store.edits != 1 {
		t.Fatalf("Edit calls = %d, want 1", store.edits)
	}

	_, reload := m.Update(msg)
	if reload == nil {
		t.Fatal("an edited document did not reload the session")
	}
	var reloaded, refetched bool
	for _, got := range drain(t, reload) {
		switch got.(type) {
		case loadedMsg:
			reloaded = true
		case detailMsg:
			refetched = true
		}
	}
	if !reloaded || !refetched {
		t.Fatalf("after an edit the session reloaded=%v refetched=%v, want both", reloaded, refetched)
	}
	if store.lists != 1 {
		t.Fatalf("List calls after the edit = %d, want 1", store.lists)
	}
}

// drain runs a command and returns the messages it produces, flattening a
// BatchMsg into its parts.
func drain(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, part := range batch {
			if part != nil {
				msgs = append(msgs, part())
			}
		}
		return msgs
	}
	return []tea.Msg{msg}
}
