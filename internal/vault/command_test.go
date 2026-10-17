package vault

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/petsagouris/specshelf/internal/frontmatter"
)

// Create derives the id and file name from the title and today's date, seeds the
// body from the kind's template, and leaves the index following the document.
func TestCreateComposesDocument(t *testing.T) {
	root := t.TempDir()
	path, err := Create(root, NewDocument{
		Kind:     "spec",
		Title:    "  Quality   analysis display contract ",
		Repos:    []string{"a", "b"},
		Tags:     []string{"c"},
		Sessions: []string{"s1"},
		Status:   "active",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	date := localDate(time.Now())
	wantID := documentID("spec", date, "quality-analysis-display-contract")
	wantPath := documentPath(filepath.Join(root, "specs"), date, "quality-analysis-display-contract")
	if path != wantPath {
		t.Errorf("path = %q, want %q", path, wantPath)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := frontmatter.Parse(path, data)
	if err != nil {
		t.Fatalf("parse created document: %v", err)
	}
	if doc.ID() != wantID {
		t.Errorf("id = %q, want %q", doc.ID(), wantID)
	}
	if doc.Title() != "Quality analysis display contract" {
		t.Errorf("title = %q, want the whitespace-normalised title", doc.Title())
	}
	if got := doc.Repos(); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("repos = %v, want [a b]", got)
	}
	if !strings.Contains(doc.Body, "# Quality analysis display contract") {
		t.Errorf("body = %q, want the seeded heading", doc.Body)
	}

	records, err := Index(root)
	if err != nil {
		t.Fatalf("index after create: %v", err)
	}
	if len(records) != 1 || records[0].ID != wantID {
		t.Errorf("index = %+v, want the created document", records)
	}
}

// A second document of the same title on the same day gets a -2 suffix on both
// its id and its file name, and neither creation is overwritten.
func TestCreateSuffixesIDAndPathTogether(t *testing.T) {
	root := t.TempDir()
	first, err := Create(root, NewDocument{Kind: "spec", Title: "Same title", Status: "active"})
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	second, err := Create(root, NewDocument{Kind: "spec", Title: "Same title", Status: "active"})
	if err != nil {
		t.Fatalf("second Create: %v", err)
	}

	firstStem := strings.TrimSuffix(first, ".md")
	if want := firstStem + "-2.md"; second != want {
		t.Errorf("second path = %q, want %q", second, want)
	}
	if firstID, secondID := readID(t, first), readID(t, second); secondID != firstID+"-2" {
		t.Errorf("second id = %q, want %q", secondID, firstID+"-2")
	}
}

// LinkSessions appends to the existing sessions, stamps updated and refreshes the
// index, keeping the document's other fields.
func TestLinkSessionsMergesAndRefreshes(t *testing.T) {
	root := t.TempDir()
	path, err := Create(root, NewDocument{
		Kind: "spec", Title: "Linked", Sessions: []string{"a"}, Status: "active",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := LinkSessions(root, "spec-"+compactDate(localDate(time.Now()))+"-linked", []string{"b", "a"})
	if err != nil {
		t.Fatalf("LinkSessions: %v", err)
	}
	if got != path {
		t.Errorf("LinkSessions returned %q, want %q", got, path)
	}
	doc, err := readDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	if sessions := doc.CodexSessions(); len(sessions) != 2 || sessions[0] != "a" || sessions[1] != "b" {
		t.Errorf("sessions = %v, want [a b]", sessions)
	}
	if doc.Updated() != localDate(time.Now()) {
		t.Errorf("updated = %q, want today", doc.Updated())
	}
	records, err := Index(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || len(records[0].CodexSessions) != 2 {
		t.Errorf("index = %+v, want the linked sessions", records)
	}

	if _, err := LinkSessions(root, doc.ID(), nil); err == nil {
		t.Error("LinkSessions with no sessions succeeded, want an error")
	}
}

// fakeEditor stands in for a user's editor in the store tests: appendText is
// written onto the document, so an empty appendText is an editor that changed
// nothing.
type fakeEditor struct{ appendText string }

func (e fakeEditor) Edit(path string) error {
	if e.appendText == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(e.appendText); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// An editor that changes nothing must leave both the document and the index
// byte-for-byte as they were: updated moves only when the content did. The
// document's dated fields are deliberately stale, so a bump would show.
func TestEditLeavesAnUnchangedDocumentAlone(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "specs", "2026-10-17-note.md")
	writeDoc(t, root, "specs/2026-10-17-note.md",
		"---\nid: spec-20261017-note\nkind: spec\ntitle: Note\nstatus: active\ncreated: 2000-01-01\nupdated: 2000-01-01\n---\n# Note\n")
	if err := WriteIndex(root); err != nil {
		t.Fatal(err)
	}
	before := mustRead(t, path)
	indexBefore := mustRead(t, filepath.Join(root, "index.jsonl"))

	got, err := Edit(root, "spec-20261017-note", fakeEditor{})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if got != path {
		t.Errorf("Edit returned %q, want %q", got, path)
	}
	if after := mustRead(t, path); !bytes.Equal(after, before) {
		t.Errorf("document changed:\n got %q\nwant %q", after, before)
	}
	if after := mustRead(t, filepath.Join(root, "index.jsonl")); !bytes.Equal(after, indexBefore) {
		t.Errorf("index changed:\n got %q\nwant %q", after, indexBefore)
	}
}

// A changed document is stamped with today's date and the index follows, while
// the editor's body and the document's unknown keys survive the rewrite.
func TestEditStampsChangedDocumentAndRefreshesIndex(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-note.md",
		"---\nid: spec-20261017-note\nkind: spec\ntitle: Note\nstatus: active\ncreated: 2000-01-01\nupdated: 2000-01-01\npriority: high\n---\n# Note\n")

	got, err := Edit(root, "spec-20261017-note", fakeEditor{appendText: "\n\nchanged by the editor\n"})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if want := filepath.Join(root, "specs", "2026-10-17-note.md"); got != want {
		t.Errorf("Edit returned %q, want %q", got, want)
	}

	doc, err := readDocument(got)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Updated() != localDate(time.Now()) {
		t.Errorf("updated = %q, want today %q", doc.Updated(), localDate(time.Now()))
	}
	if doc.Created() != "2000-01-01" {
		t.Errorf("created = %q, want it untouched", doc.Created())
	}
	if doc.Title() != "Note" || doc.Status() != "active" || doc.Kind() != "spec" {
		t.Errorf("rewrite changed id/kind/title/status: %+v", doc.Metadata)
	}
	if !strings.Contains(doc.Body, "changed by the editor") {
		t.Errorf("body = %q, want the editor's text", doc.Body)
	}
	if doc.Metadata["priority"] != "high" {
		t.Errorf("unknown key priority = %v, want it preserved", doc.Metadata["priority"])
	}
	records, err := Index(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || !strings.Contains(records[0].Summary, "changed by the editor") {
		t.Errorf("index = %+v, want the edited body's summary", records)
	}
}

// A failing editor leaves the document as it was.
func TestEditLeavesDocumentWhenTheEditorFails(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-note.md",
		"---\nid: spec-20261017-note\nkind: spec\ntitle: Note\nstatus: active\ncreated: 2000-01-01\nupdated: 2000-01-01\n---\n# Note\n")
	before := mustRead(t, filepath.Join(root, "specs", "2026-10-17-note.md"))

	if _, err := Edit(root, "spec-20261017-note", errEditor{}); err == nil {
		t.Fatal("Edit succeeded although the editor failed")
	}
	if after := mustRead(t, filepath.Join(root, "specs", "2026-10-17-note.md")); !bytes.Equal(after, before) {
		t.Errorf("document changed after a failed editor:\n got %q\nwant %q", after, before)
	}
}

type errEditor struct{}

func (errEditor) Edit(string) error { return os.ErrPermission }

// SetStatus sets the frontmatter status, bumps updated to today and refreshes
// the index, leaving created and the body alone.
func TestSetStatusSetsStatusAndBumpsDate(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-note.md",
		"---\nid: spec-20261017-note\nkind: spec\ntitle: Note\nstatus: active\ncreated: 2000-01-01\nupdated: 2000-01-01\n---\n# Note\n")

	got, err := SetStatus(root, "2026-10-17-note", "accepted")
	if err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	doc, err := readDocument(got)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Status() != "accepted" {
		t.Errorf("status = %q, want accepted", doc.Status())
	}
	if doc.Updated() != localDate(time.Now()) {
		t.Errorf("updated = %q, want today %q", doc.Updated(), localDate(time.Now()))
	}
	if doc.Created() != "2000-01-01" {
		t.Errorf("created = %q, want it untouched", doc.Created())
	}
	if !strings.Contains(doc.Body, "# Note") {
		t.Errorf("body = %q, want it untouched", doc.Body)
	}
	records, err := Index(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Status != "accepted" {
		t.Errorf("index = %+v, want the new status", records)
	}
}

// Remove deletes the document and its index line, and an unknown reference
// removes nothing.
func TestRemoveDeletesAndRefreshesIndex(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-note.md",
		"---\nid: spec-20261017-note\nkind: spec\ntitle: Note\nstatus: active\ncreated: 2000-01-01\nupdated: 2000-01-01\n---\n# Note\n")
	if err := WriteIndex(root); err != nil {
		t.Fatal(err)
	}

	got, err := Remove(root, "spec-20261017-note")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if want := filepath.Join(root, "specs", "2026-10-17-note.md"); got != want {
		t.Errorf("Remove returned %q, want %q", got, want)
	}
	if _, err := os.Stat(got); err == nil {
		t.Errorf("%s still exists after Remove", got)
	}
	if bytes.Contains(mustRead(t, filepath.Join(root, "index.jsonl")), []byte("spec-20261017-note")) {
		t.Error("the removed document is still indexed")
	}

	if _, err := Remove(root, "2026-10-17-note"); err == nil {
		t.Error("Remove of an unknown reference succeeded, want an error")
	}
}

// Remove is confined to the vault. Reference resolution tries an existing path
// first, so without a check `rm <path>` deletes any file the user names; a
// resolved document outside the root is refused with a typed error and left
// alone, while one inside still deletes.
func TestRemoveConfinement(t *testing.T) {
	root := filepath.Join(t.TempDir(), "vault")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	const doc = "---\nid: spec-20261017-note\nkind: spec\ntitle: Note\n---\n# Note\n"

	t.Run("a document inside the vault still deletes", func(t *testing.T) {
		inside := filepath.Join(root, "specs", "2026-10-17-note.md")
		if err := os.MkdirAll(filepath.Dir(inside), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(inside, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Remove(root, "spec-20261017-note")
		if err != nil {
			t.Fatalf("Remove: %v", err)
		}
		if got != inside {
			t.Errorf("Remove returned %q, want %q", got, inside)
		}
		if _, err := os.Stat(inside); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s still exists after Remove: %v", inside, err)
		}
	})

	t.Run("a path outside the vault is refused and left alone", func(t *testing.T) {
		outside := filepath.Join(t.TempDir(), "outside.md")
		if err := os.WriteFile(outside, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Remove(root, outside)
		var outsideErr *OutsideVaultError
		if !errors.As(err, &outsideErr) {
			t.Fatalf("Remove error = %v, want *OutsideVaultError", err)
		}
		if outsideErr.Path != outside || !strings.Contains(err.Error(), outside) {
			t.Errorf("error %q must name the path %s", err.Error(), outside)
		}
		if !strings.Contains(err.Error(), root) {
			t.Errorf("error %q must name the root %s", err.Error(), root)
		}
		if _, statErr := os.Stat(outside); statErr != nil {
			t.Errorf("%s was removed despite being outside the vault: %v", outside, statErr)
		}
	})

	t.Run("a symlinked directory escaping the vault is refused", func(t *testing.T) {
		target := t.TempDir()
		link := filepath.Join(root, "escape")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		escaping := filepath.Join(link, "outside.md")
		if err := os.WriteFile(escaping, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Remove(root, escaping)
		var outsideErr *OutsideVaultError
		if !errors.As(err, &outsideErr) {
			t.Fatalf("Remove error = %v, want *OutsideVaultError", err)
		}
		if _, statErr := os.Stat(escaping); statErr != nil {
			t.Errorf("%s was removed through a symlinked directory: %v", escaping, statErr)
		}
	})
}

func readID(t *testing.T, path string) string {
	t.Helper()
	doc, err := readDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	return doc.ID()
}
