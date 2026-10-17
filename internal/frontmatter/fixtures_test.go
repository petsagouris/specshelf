package frontmatter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// indexRecord is the part of the reference implementation's index.jsonl that the
// reader is responsible for.
type indexRecord struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Title         string   `json:"title"`
	Status        string   `json:"status"`
	Created       string   `json:"created"`
	Updated       string   `json:"updated"`
	Repos         []string `json:"repos"`
	Tags          []string `json:"tags"`
	CodexSessions []string `json:"codex_sessions"`
	Path          string   `json:"path"`
}

// The reader's acceptance test. Every document in both reference vaults is parsed
// and compared against the index the reference implementation itself wrote for
// those documents, so the expectations are the other tool's recorded view rather
// than a restatement of our own implementation.
func TestReaderAgreesWithReferenceFixtures(t *testing.T) {
	for _, vault := range []string{"python_vault", "python_vault_rewritten"} {
		t.Run(vault, func(t *testing.T) {
			checkVaultAgainstItsIndex(t, filepath.Join("..", "..", "testdata", vault))
		})
	}
}

func checkVaultAgainstItsIndex(t *testing.T, root string) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, "index.jsonl"))
	if err != nil {
		t.Fatalf("reading the reference index: %v", err)
	}
	var records []indexRecord
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line == "" {
			continue
		}
		var record indexRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("parsing an index line: %v", err)
		}
		records = append(records, record)
	}
	if len(records) == 0 {
		t.Fatal("the reference index lists no documents; the fixtures are not what this test expects")
	}

	for _, want := range records {
		data, err := os.ReadFile(filepath.Join(root, want.Path))
		if err != nil {
			t.Fatalf("reading %s: %v", want.Path, err)
		}
		doc, err := Parse(want.Path, data)
		if err != nil {
			t.Errorf("%s: Parse: %v", want.Path, err)
			continue
		}

		if got := doc.ID(); got != want.ID {
			t.Errorf("%s: ID() = %q, want %q", want.Path, got, want.ID)
		}
		if got := doc.Kind(); got != want.Kind {
			t.Errorf("%s: Kind() = %q, want %q", want.Path, got, want.Kind)
		}
		if got := doc.Status(); got != want.Status {
			t.Errorf("%s: Status() = %q, want %q", want.Path, got, want.Status)
		}
		if got := doc.Created(); got != want.Created {
			t.Errorf("%s: Created() = %q, want %q", want.Path, got, want.Created)
		}
		if got := doc.Updated(); got != want.Updated {
			t.Errorf("%s: Updated() = %q, want %q", want.Path, got, want.Updated)
		}
		if got := doc.Repos(); !sameStrings(got, want.Repos) {
			t.Errorf("%s: Repos() = %q, want %q", want.Path, got, want.Repos)
		}
		if got := doc.Tags(); !sameStrings(got, want.Tags) {
			t.Errorf("%s: Tags() = %q, want %q", want.Path, got, want.Tags)
		}
		if got := doc.CodexSessions(); !sameStrings(got, want.CodexSessions) {
			t.Errorf("%s: CodexSessions() = %q, want %q", want.Path, got, want.CodexSessions)
		}

		// The one documented divergence: the reference parser reads an empty
		// value as an empty list and records the title as "[]". We record it as
		// empty, so a document with no title keeps no title instead of decaying
		// into the text "[]" the next time it is written.
		wantTitle := want.Title
		if wantTitle == "[]" {
			wantTitle = ""
		}
		if got := doc.Title(); got != wantTitle {
			t.Errorf("%s: Title() = %q, want %q", want.Path, got, wantTitle)
		}
		if doc.Body == "" {
			t.Errorf("%s: the body came back empty", want.Path)
		}
	}

	// The index is a complete account of the documents on disk, so a stale or
	// short index cannot make this test pass vacuously.
	onDisk := 0
	for _, dir := range []string{"specs", "decisions"} {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			t.Fatalf("reading %s: %v", dir, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				onDisk++
			}
		}
	}
	if onDisk != len(records) {
		t.Errorf("%d documents on disk but %d in the index", onDisk, len(records))
	}
}

// The reference tool writes titles that a YAML parser misreads — a leading hash
// becomes a comment, a leading dash a sequence entry, an asterisk an alias. They
// have to come back byte for byte, or an existing vault is unreadable.
func TestReferenceFixturesKeepInvalidYAMLTitles(t *testing.T) {
	titles := map[string]string{
		"2026-10-17-fix-42-the-thing": "Fix #42: the thing",
		"2026-10-17-1-priority":       "#1 priority",
		"2026-10-17-dash-start":       "- dash start",
		"2026-10-17-bold-start":       "*bold* start",
		"2026-10-17-he-said-hi":       `He said "hi"`,
		"2026-10-17-trailing-space":   "Trailing space",
	}
	for stem, want := range titles {
		name := filepath.Join("specs", stem+".md")
		data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "python_vault", name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		doc, err := Parse(name, data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := doc.Title(); got != want {
			t.Errorf("%s: Title() = %q, want %q", stem, got, want)
		}
	}
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
