package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/petsagouris/specshelf/internal/vault"
)

// requestDelete asks the session to delete the open document. 5.5 turns this
// into a confirmation step.
func requestDelete(m *Model) tea.Cmd { return update(m, k('d')) }

// withoutLine returns index with the line containing needle removed, so a test
// can assert the index changed only by losing the deleted document's record.
func withoutLine(index, needle string) string {
	var kept []string
	for _, line := range strings.Split(strings.TrimSuffix(index, "\n"), "\n") {
		if strings.Contains(line, needle) {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n") + "\n"
}

// TestSessionCreateByteIdenticalToCommandLine is the 5.4 byte-identical test: a
// document created in the session is byte-for-byte what `specshelf new` writes
// for the same title and kind, because both go through vault.Create.
func TestSessionCreateByteIdenticalToCommandLine(t *testing.T) {
	in := vault.NewDocument{Kind: "spec", Title: "Parity spec"}

	sessionRoot := t.TempDir()
	m := New(VaultStore{Root: sessionRoot})
	update(m, execCmd(t, m.Init()))
	update(m, k('c'))
	for _, r := range in.Title {
		update(m, k(r))
	}
	for _, msg := range drain(t, update(m, kc(tea.KeyEnter))) {
		update(m, msg)
	}

	records, _, err := vault.List(sessionRoot)
	if err != nil {
		t.Fatalf("list after session create: %v", err)
	}
	var sessionPath string
	for _, record := range records {
		if record.Title == in.Title {
			sessionPath = filepath.Join(sessionRoot, record.Path)
		}
	}
	if sessionPath == "" {
		t.Fatalf("the session did not create %q: %+v", in.Title, records)
	}
	sessionBytes, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("read session document: %v", err)
	}

	cliRoot := t.TempDir()
	cliPath, err := vault.Create(cliRoot, in)
	if err != nil {
		t.Fatalf("command-line create: %v", err)
	}
	cliBytes, err := os.ReadFile(cliPath)
	if err != nil {
		t.Fatalf("read command-line document: %v", err)
	}

	if !bytes.Equal(sessionBytes, cliBytes) {
		t.Fatalf("session and command-line documents differ\nsession:\n%s\ncommand line:\n%s", sessionBytes, cliBytes)
	}
}

// TestIndexConsistentAfterSessionDelete is the 5.4 index test: deleting in the
// session drops the document's record and leaves every other record untouched.
func TestIndexConsistentAfterSessionDelete(t *testing.T) {
	root := t.TempDir()
	for _, title := range []string{"Alpha spec", "Beta spec"} {
		if _, err := vault.Create(root, vault.NewDocument{Kind: "spec", Title: title}); err != nil {
			t.Fatalf("seed %q: %v", title, err)
		}
	}

	m := New(VaultStore{Root: root})
	update(m, execCmd(t, m.Init()))
	visible := m.visible()
	if len(visible) != 2 {
		t.Fatalf("visible documents = %d, want 2", len(visible))
	}
	deleted, kept := visible[0].ID, visible[1].ID

	indexPath := filepath.Join(root, "index.jsonl")
	before, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index before: %v", err)
	}

	update(m, execCmd(t, update(m, kc(tea.KeyEnter))))
	for _, msg := range drain(t, requestDelete(m)) {
		update(m, msg)
	}

	after, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index after: %v", err)
	}
	if want := withoutLine(string(before), `"id": "`+deleted+`"`); string(after) != want {
		t.Fatalf("the index changed by more than the deleted record\nwant:\n%s\ngot:\n%s", want, after)
	}
	if !strings.Contains(string(after), `"id": "`+kept+`"`) {
		t.Fatalf("the surviving document left the index:\n%s", after)
	}
	if _, err := vault.Find(root, deleted); err == nil {
		t.Fatal("the deleted document is still readable")
	}
}

// TestCreateKeyRoutesThroughStore is 5.4 at the session level: the create prompt
// submits through the store and reloads the list.
func TestCreateKeyRoutesThroughStore(t *testing.T) {
	m, store := newFake()
	update(m, k('c'))
	for _, r := range "Brand new" {
		update(m, k(r))
	}
	msg := execCmd(t, update(m, kc(tea.KeyEnter)))
	if store.creates != 1 {
		t.Fatalf("Create calls = %d, want 1", store.creates)
	}
	if store.lastCreate.Kind != "spec" || store.lastCreate.Title != "Brand new" {
		t.Fatalf("create request = %+v, want kind spec title %q", store.lastCreate, "Brand new")
	}
	_, reload := m.Update(msg)
	if reload == nil {
		t.Fatal("a created document did not reload the list")
	}
}

// TestStatusChangeRoutesThroughStore is 5.4 for the status path: the prompt
// applies the typed status through the store and reloads.
func TestStatusChangeRoutesThroughStore(t *testing.T) {
	m, store := newFake()
	update(m, execCmd(t, update(m, kc(tea.KeyEnter))))
	update(m, k('s'))
	if m.mode != modeStatus {
		t.Fatalf("mode = %d, want status", m.mode)
	}
	for _, r := range "done" {
		update(m, k(r))
	}
	msg := execCmd(t, update(m, kc(tea.KeyEnter)))
	if store.statusCalls != 1 {
		t.Fatalf("SetStatus calls = %d, want 1", store.statusCalls)
	}
	if store.lastStatus != "done" {
		t.Fatalf("status = %q, want %q", store.lastStatus, "done")
	}
	if _, reload := m.Update(msg); reload == nil {
		t.Fatal("a status change did not reload the session")
	}
}
