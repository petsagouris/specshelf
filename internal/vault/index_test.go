package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	fixtureEmptyTitle = `"title": "[]"`
	ourEmptyTitle     = `"title": ""`
)

// readFixture reads a reference file, failing the test when it cannot.
func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return data
}

// assertEmptyTitleDivergence pins the one intentional difference: the reference
// reads the empty-title document's `title:` back as an empty list and records
// the text "[]", while our reader reads it as the empty scalar (task 2.3). The
// assertion fails if a future fixture stops exercising it, so the substitution
// below can never hide another difference.
func assertEmptyTitleDivergence(t *testing.T, fixture []byte) {
	t.Helper()
	if n := bytes.Count(fixture, []byte(fixtureEmptyTitle)); n != 1 {
		t.Fatalf("fixture contains %d occurrences of %s, want exactly 1", n, fixtureEmptyTitle)
	}
}

func TestIndexReferenceVaultParity(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "python_vault")
	records, err := Index(root)
	if err != nil {
		t.Fatalf("Index(%q): %v", root, err)
	}
	if len(records) != 10 {
		t.Fatalf("Index(%q) returned %d records, want 10", root, len(records))
	}

	want := readFixture(t, filepath.Join(root, "index.jsonl"))
	assertEmptyTitleDivergence(t, want)
	want = bytes.ReplaceAll(want, []byte(fixtureEmptyTitle), []byte(ourEmptyTitle))

	if got := EncodeIndex(records); !bytes.Equal(got, want) {
		t.Errorf("index bytes differ from the reference:\n got %q\nwant %q", got, want)
	}
}

func TestIndexRewrittenVaultParity(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "python_vault_rewritten")
	records, err := Index(root)
	if err != nil {
		t.Fatalf("Index(%q): %v", root, err)
	}
	if len(records) != 10 {
		t.Fatalf("Index(%q) returned %d records, want 10", root, len(records))
	}

	want := readFixture(t, filepath.Join(root, "index.jsonl"))
	assertEmptyTitleDivergence(t, want)
	want = bytes.ReplaceAll(want, []byte(fixtureEmptyTitle), []byte(ourEmptyTitle))

	if got := EncodeIndex(records); !bytes.Equal(got, want) {
		t.Errorf("rewritten index bytes differ from the reference:\n got %q\nwant %q", got, want)
	}
}

// writeDoc writes a document at a vault-relative slash path.
func writeDoc(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIndexTemporaryVault(t *testing.T) {
	t.Run("empty vault", func(t *testing.T) {
		root := t.TempDir()
		records, err := Index(root)
		if err != nil {
			t.Fatalf("Index: %v", err)
		}
		if len(records) != 0 {
			t.Fatalf("got %d records, want 0", len(records))
		}
		if got := EncodeIndex(records); len(got) != 0 {
			t.Errorf("EncodeIndex = %q, want empty", got)
		}
	})

	t.Run("missing decisions directory", func(t *testing.T) {
		root := t.TempDir()
		writeDoc(t, root, "specs/2026-10-17-only-spec.md", "---\nid: spec-20261017-only-spec\nkind: spec\ntitle: Only spec\n---\n# Only spec\n")
		records, err := Index(root)
		if err != nil {
			t.Fatalf("Index: %v", err)
		}
		if len(records) != 1 || records[0].ID != "spec-20261017-only-spec" {
			t.Fatalf("got %+v, want the one spec", records)
		}
	})

	t.Run("specs before decisions", func(t *testing.T) {
		root := t.TempDir()
		// The decision's file name sorts before the spec's, so the assertion
		// fails unless the directory, not the name, decides the order.
		writeDoc(t, root, "specs/z-spec.md", "---\nid: spec-20261017-z\nkind: spec\ntitle: Z\n---\n# Z\n")
		writeDoc(t, root, "decisions/a-decision.md", "---\nid: decision-20261017-a\nkind: decision\ntitle: A\n---\n# A\n")
		records, err := Index(root)
		if err != nil {
			t.Fatalf("Index: %v", err)
		}
		if len(records) != 2 || records[0].Kind != "spec" || records[1].Kind != "decision" {
			t.Fatalf("got %+v, want the spec before the decision", records)
		}
	})

	t.Run("summary is the first non-empty non-heading line", func(t *testing.T) {
		root := t.TempDir()
		const body = "# Summary test\n\n   # an indented heading is still a heading\n\n   The real summary line.  \n\n## Goals\n"
		writeDoc(t, root, "specs/2026-10-17-summary.md", "---\nid: spec-20261017-summary\nkind: spec\ntitle: Summary test\n---\n"+body)
		records, err := Index(root)
		if err != nil {
			t.Fatalf("Index: %v", err)
		}
		if len(records) != 1 {
			t.Fatalf("got %d records, want 1", len(records))
		}
		if records[0].Summary != "The real summary line." {
			t.Errorf("Summary = %q, want %q", records[0].Summary, "The real summary line.")
		}
	})

	t.Run("non-ASCII title is literal", func(t *testing.T) {
		root := t.TempDir()
		const title = "质量分析展示契约"
		writeDoc(t, root, "specs/2026-10-17-cjk.md", "---\nid: spec-20261017-cjk\nkind: spec\ntitle: "+title+"\n---\n# "+title+"\n")
		records, err := Index(root)
		if err != nil {
			t.Fatalf("Index: %v", err)
		}
		if len(records) != 1 || records[0].Title != title {
			t.Fatalf("got %+v, want title %q", records, title)
		}
		encoded := EncodeIndex(records)
		if !bytes.Contains(encoded, []byte(title)) || bytes.Contains(encoded, []byte(`\u`)) {
			t.Errorf("EncodeIndex = %q, want the literal UTF-8 title", encoded)
		}
	})

	t.Run("empty list field and unknown key", func(t *testing.T) {
		root := t.TempDir()
		writeDoc(t, root, "specs/2026-10-17-keys.md", "---\nid: spec-20261017-keys\nkind: spec\ntitle: Keys\ntags:\n  - one\nrepos:\nx_custom: yes\n---\n# Keys\n")
		records, err := Index(root)
		if err != nil {
			t.Fatalf("Index: %v", err)
		}
		if len(records) != 1 {
			t.Fatalf("got %d records, want 1", len(records))
		}
		encoded := EncodeIndex(records)
		if !bytes.Contains(encoded, []byte(`"repos": []`)) {
			t.Errorf("EncodeIndex = %q, want the empty list field as []", encoded)
		}
		if !bytes.Contains(encoded, []byte(`"tags": ["one"]`)) {
			t.Errorf("EncodeIndex = %q, want tags [\"one\"]", encoded)
		}
		if bytes.Contains(encoded, []byte("x_custom")) {
			t.Errorf("EncodeIndex = %q, must not carry the unknown key", encoded)
		}
		if !strings.HasSuffix(string(encoded), "\n") || strings.Count(string(encoded), "\n") != 1 {
			t.Errorf("EncodeIndex = %q, want one newline-terminated line", encoded)
		}
	})
}

// The summary splits on Python's str.splitlines breaks, not just \n: a body
// holding U+2028, a vertical tab or another exotic separator summarises the way
// the reference implementation does, not as one long line.
func TestSummarySplitsOnPythonLineBreaks(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"newline", "first\nsecond", "first"},
		{"carriage return", "first\rsecond", "first"},
		{"vertical tab", "first\vsecond", "first"},
		{"form feed", "first\fsecond", "first"},
		{"record separator", "first\x1esecond", "first"},
		{"next line", "first\u0085second", "first"},
		{"line separator", "first\u2028second", "first"},
		{"paragraph separator", "first\u2029second", "first"},
		{"heading before a line separator", "# Heading\u2028the real summary", "the real summary"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := summary(c.body); got != c.want {
				t.Errorf("summary(%q) = %q, want %q", c.body, got, c.want)
			}
		})
	}
}

// TestIndexPathUsesForwardSlashes pins the separator divergence: an index path
// is vault-relative with "/" on every platform, the way the reference writes it,
// never the platform separator (task 3.13). The platform join differs from "/"
// only where filepath.Separator does, so on Unix this test and the fixture-parity
// tests agree by construction; on Windows a revert to filepath.Join fails here.
func TestIndexPathUsesForwardSlashes(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-slash.md", "---\nid: spec-20261017-slash\nkind: spec\ntitle: Slash\n---\n# Slash\n")

	records, err := Index(root)
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if got, want := records[0].Path, "specs/2026-10-17-slash.md"; got != want {
		t.Errorf("record path = %q, want %q", got, want)
	}
	if strings.ContainsRune(records[0].Path, '\\') {
		t.Errorf("record path %q carries a backslash separator", records[0].Path)
	}
	if got := filepath.ToSlash(records[0].Path); got != records[0].Path {
		t.Errorf("record path %q is not slash-separated on this platform", records[0].Path)
	}
}

// Index must ignore anything in the document directories that is not Markdown:
// another file type, a file with no extension, and a directory whose name ends
// in .md.
func TestIndexIgnoresNonMarkdownEntry(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-kept.md", "---\nid: spec-20261017-kept\nkind: spec\ntitle: Kept\n---\n# Kept\n")
	writeDoc(t, root, "specs/notes.txt", "not a document\n")
	writeDoc(t, root, "specs/README", "no extension\n")
	if err := os.Mkdir(filepath.Join(root, "specs", "nested.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	records, err := Index(root)
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	if len(records) != 1 || records[0].Path != "specs/2026-10-17-kept.md" {
		t.Errorf("records = %+v, want only the Markdown document", records)
	}
}

// Index is the strict path: a document it cannot read is an error naming the
// file, unlike the tolerant List.
func TestUnreadableDocumentFailsIndex(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-broken.md", "---\nid: spec-20261017-broken\nkind: spec\n")

	_, err := Index(root)
	if err == nil {
		t.Fatal("Index succeeded on an unterminated document, want an error")
	}
	if !strings.Contains(err.Error(), "specs/2026-10-17-broken.md") {
		t.Errorf("Index error %q does not name the failing file", err)
	}
}

func TestIndexJSONEscaping(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"quote and backslash", `a"b\c`, `"a\"b\\c"`},
		{"short control escapes", "a\b\f\n\r\tz", `"a\b\f\n\r\tz"`},
		{"other control characters", "\x00\x01\x1f", `"\u0000\u0001\u001f"`},
		{"delete is literal", "\x7f", "\"\x7f\""},
		{"non-ASCII and JSON-hostile characters are literal", "<&>\u2028\u2029质量", "\"<&>\u2028\u2029质量\""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := string(appendJSONString(nil, c.in)); got != c.want {
				t.Errorf("appendJSONString(%q) = %s, want %s", c.in, got, c.want)
			}
		})
	}
}
