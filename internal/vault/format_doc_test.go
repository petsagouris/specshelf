package vault

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/petsagouris/specshelf/internal/frontmatter"
)

// TestFormatDocSample parses the document sample in docs/FORMAT.md with the real
// reader, so the documented format and the code cannot drift. The sample is
// read from the file rather than copied here: editing it changes what the test
// checks.
func TestFormatDocSample(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "FORMAT.md"))
	if err != nil {
		t.Fatalf("read docs/FORMAT.md: %v", err)
	}
	sample := fencedBlock(t, string(data), "markdown")

	doc, err := frontmatter.Parse("FORMAT.md", []byte(sample))
	if err != nil {
		t.Fatalf("parse the FORMAT.md sample: %v", err)
	}

	if got, want := doc.ID(), "spec-20261017-quality-analysis-display-contract"; got != want {
		t.Errorf("id = %q, want %q", got, want)
	}
	if got, want := doc.Kind(), "spec"; got != want {
		t.Errorf("kind = %q, want %q", got, want)
	}
	if got, want := doc.Title(), "Quality analysis display contract"; got != want {
		t.Errorf("title = %q, want %q", got, want)
	}
	if got, want := doc.Status(), "active"; got != want {
		t.Errorf("status = %q, want %q", got, want)
	}
	if got, want := doc.Created(), "2026-10-17"; got != want {
		t.Errorf("created = %q, want %q", got, want)
	}
	if got, want := doc.Updated(), "2026-10-18"; got != want {
		t.Errorf("updated = %q, want %q", got, want)
	}
	if got, want := doc.Repos(), []string{"lippi-smart-customer", "aics-web-repos"}; !slices.Equal(got, want) {
		t.Errorf("repos = %q, want %q", got, want)
	}
	if got, want := doc.Tags(), []string{"quality-analysis", "contract"}; !slices.Equal(got, want) {
		t.Errorf("tags = %q, want %q", got, want)
	}
	if got, want := doc.CodexSessions(), []string{"session-a", "session-b"}; !slices.Equal(got, want) {
		t.Errorf("codex_sessions = %q, want %q", got, want)
	}

	heading := "# " + doc.Title()
	if first, _, _ := strings.Cut(doc.Body, "\n"); first != heading {
		t.Errorf("body heading = %q, want %q", first, heading)
	}

	// The sample is written the way the writer writes, so re-encoding the parsed
	// document must reproduce it — this is what binds the documented field order
	// and quoting to the codec.
	if got := string(frontmatter.Marshal(doc)); got != sample {
		t.Errorf("re-encoded sample differs:\n got %q\nwant %q", got, sample)
	}
}

// fencedBlock returns the content of the first fenced block in doc whose info
// string is info.
func fencedBlock(t *testing.T, doc, info string) string {
	t.Helper()
	open := "```" + info + "\n"
	start := strings.Index(doc, open)
	if start == -1 {
		t.Fatalf("docs/FORMAT.md has no %s fenced block", info)
	}
	rest := doc[start+len(open):]
	end := strings.Index(rest, "\n```")
	if end == -1 {
		t.Fatalf("docs/FORMAT.md has an unterminated %s fenced block", info)
	}
	return rest[:end+1]
}
