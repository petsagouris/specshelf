package vault

import (
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

func readID(t *testing.T, path string) string {
	t.Helper()
	doc, err := readDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	return doc.ID()
}
