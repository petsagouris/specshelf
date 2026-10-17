package vault

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/petsagouris/specshelf/internal/frontmatter"
)

// A listing skips a document it cannot read, warns once with the path and the
// reason, and carries on with the readable documents. The strict Index path
// still fails on the same document, which is what `doctor` will build on.
func TestSkipUnreadableDocumentWarns(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-good.md", "---\nid: spec-20261017-good\nkind: spec\ntitle: Good\n---\n# Good\n\nBody.\n")
	writeDoc(t, root, "specs/2026-10-17-broken.md", "---\nid: spec-20261017-broken\nkind: spec\ntitle: Broken\n")
	writeDoc(t, root, "specs/2026-10-17-also-good.md", "---\nid: spec-20261017-also-good\nkind: spec\ntitle: Also good\n---\n# Also good\n\nBody.\n")

	records, warnings, err := List(root)
	if err != nil {
		t.Fatalf("List returned a hard error for an unreadable document: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("List returned %d records, want the 2 readable documents", len(records))
	}
	if records[0].ID != "spec-20261017-also-good" || records[1].ID != "spec-20261017-good" {
		t.Errorf("records = %+v, want the readable documents in index order", records)
	}

	if len(warnings) != 1 {
		t.Fatalf("List returned %d warnings, want exactly one for the one bad file: %v", len(warnings), warnings)
	}
	warning := warnings[0]
	if warning.Path != "specs/2026-10-17-broken.md" {
		t.Errorf("warning path = %q, want the vault relative path", warning.Path)
	}
	if !strings.Contains(warning.Reason, "unterminated frontmatter") {
		t.Errorf("warning reason = %q, want it to name the parse error", warning.Reason)
	}
	text := warning.Text()
	if !strings.Contains(text, warning.Path) || !strings.Contains(text, warning.Reason) {
		t.Errorf("warning text = %q, want it to name both the path and the reason", text)
	}
	if strings.Contains(text, "specshelf:") {
		t.Errorf("warning text = %q, must have no CLI prefix; that belongs to the command surface", text)
	}

	// The strict path is unchanged: the same document fails Index, naming the file.
	if _, err := Index(root); err == nil {
		t.Fatal("Index succeeded on an unreadable document, want an error")
	} else if !strings.Contains(err.Error(), "specs/2026-10-17-broken.md") {
		t.Errorf("Index error %q does not name the failing file", err)
	}
}

func TestUnreadableReasonExtraction(t *testing.T) {
	parseErr := fmt.Errorf("%w: %s", frontmatter.ErrUnterminated, "specs/x.md")
	if w := warn("specs/x.md", parseErr); w.Reason != "unterminated frontmatter" {
		t.Errorf("parse reason = %q, want the reason without the path", w.Reason)
	}
	readErr := &fs.PathError{Op: "open", Path: "/vault/specs/x.md", Err: fs.ErrPermission}
	if w := warn("specs/x.md", readErr); w.Reason != fs.ErrPermission.Error() {
		t.Errorf("read reason = %q, want the OS message %q", w.Reason, fs.ErrPermission.Error())
	}
}

func TestSkipWarnsOncePerUnreadableDocument(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-a.md", "no frontmatter, no error\n")
	writeDoc(t, root, "decisions/2026-10-17-b.md", "---\nid: decision-20261017-b\n")
	writeDoc(t, root, "decisions/2026-10-17-c.md", "---\nid: decision-20261017-c\n")

	records, warnings, err := List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 1 || records[0].ID != "2026-10-17-a" {
		t.Errorf("records = %+v, want only the body-only document", records)
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings = %v, want one per unreadable document", warnings)
	}
	for i, want := range []string{"decisions/2026-10-17-b.md", "decisions/2026-10-17-c.md"} {
		if warnings[i].Path != want {
			t.Errorf("warning %d path = %q, want %q", i, warnings[i].Path, want)
		}
	}
}
